package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"

	"github.com/corona10/goimagehash"
)

type imagecheck struct {
	name string
	dh   *goimagehash.ImageHash
}

func (ic *imagecheck) String() string {
	return ic.name
}

func main() {
	tht := flag.Uint("t", 5, "duplicate throttle, max is 64")
	dir := flag.String("d", "./", "work directory")
	a := flag.Bool("a", false, "action sort")
	h := flag.Bool("h", false, "display help")
	s := flag.Uint("s", 0, "folder sequence number start (exclude)")
	flag.Parse()
	if *h {
		fmt.Println("Usage:", os.Args[0], "[-adht] ext1 ext2...")
		flag.PrintDefaults()
		fmt.Println("  exts\tmatching extensions")
		os.Exit(0)
	}
	throttle := *tht
	if throttle > 64 {
		panic("invalid throttle")
	}
	exts := flag.Args()
	for i, e := range exts {
		exts[i] = strings.ToLower(e)
	}
	fmt.Println("match extension:", exts)
	err := os.Chdir(*dir)
	if err != nil {
		panic(err)
	}
	imgs, err := os.ReadDir("./")
	if err != nil {
		panic(err)
	}
	action := *a
	chklst := make([]imagecheck, 0, len(imgs))
	fmt.Println("read", len(imgs), "files...")
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	part := len(imgs) / runtime.NumCPU()
	wg.Add(runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		from := i * part
		to := (i + 1) * part
		if to > len(imgs) {
			to = len(imgs)
		}
		isextinlist := func(n string) bool {
			for _, e := range exts {
				if strings.HasSuffix(strings.ToLower(n), e) {
					return true
				}
			}
			return false
		}
		go func(from, to int) {
			for i := from; i < to; i++ {
				img := imgs[i]
				n := img.Name()
				if !img.IsDir() && isextinlist(n) {
					f, err := os.Open(n)
					if err != nil {
						fmt.Println("ERROR:", err)
						continue
					}
					im, _, err := image.Decode(f)
					_ = f.Close()
					if err != nil {
						fmt.Println("ERROR:", err)
						continue
					}
					dh, err := goimagehash.DifferenceHash(im)
					if err != nil {
						fmt.Println("ERROR:", err)
						continue
					}
					mu.Lock()
					chklst = append(chklst, imagecheck{
						name: n,
						dh:   dh,
					})
					fmt.Print("read: ", len(chklst), " / ", len(imgs), "\r")
					mu.Unlock()
				}
			}
			wg.Done()
		}(from, to)
	}
	wg.Wait()
	fmt.Println("read file success, comparing...")
	duplis := make(map[string]uint, len(chklst))
	sameset := make([][]uint, 0)
	for i := 0; i < len(chklst); i++ {
		fmt.Print("compare: ", i, " / ", len(chklst), "\r")
		_, ok := duplis[chklst[i].name]
		if ok {
			continue
		}
		isfirst := true
		for j := len(chklst) - 1; j > i; j-- {
			dis, err := chklst[i].dh.Distance(chklst[j].dh)
			if err != nil {
				fmt.Println("ERROR:", err)
				continue
			}
			if uint(dis) < throttle {
				x, ok := duplis[chklst[j].name]
				if ok {
				LOP:
					for k, set := range sameset {
						for _, item := range set {
							if x == item {
								if isfirst {
									sameset[k] = append(sameset[k], uint(i))
									duplis[chklst[i].name] = uint(i)
									isfirst = false
								} else {
								INNERLOP:
									for l, set := range sameset {
										for _, item := range set {
											if item == uint(i) {
												sameset[k] = append(sameset[k], set...)
												sameset = append(sameset[:l], sameset[l+1:]...)
												break INNERLOP
											}
										}
									}
								}
								break LOP
							}
						}
					}
				} else if isfirst {
					sameset = append(sameset, []uint{uint(i)})
					duplis[chklst[i].name] = uint(i)
					isfirst = false
				}
				duplis[chklst[j].name] = uint(i)
			}
		}
	}
	fmt.Println("compare file success")
	if len(sameset) > 0 {
		dupset := make(map[uint][]string, len(sameset))
		setindex := func(i uint) uint {
			for _, set := range sameset {
				for _, n := range set {
					if n == i {
						return set[0]
					}
				}
			}
			panic("internal logic error")
		}
		for k, v := range duplis {
			i := setindex(v)
			dupset[i] = append(dupset[i], k)
		}
		j := *s
		for _, lst := range dupset {
			if len(lst) > 0 {
				j++
				fmt.Println("[", j, "] duplicate:", lst)
				if action {
					newdir := strconv.FormatUint(uint64(j), 10)
					err = os.MkdirAll(newdir, 0755)
					if err != nil {
						fmt.Println("ERROR:", err)
						continue
					}
					for _, i := range lst {
						err = os.Rename(i, newdir+"/"+i)
						if err != nil {
							fmt.Println("ERROR:", err)
						}
					}
				}
			}
		}
	} else {
		fmt.Println("no duplicated file")
	}
}
