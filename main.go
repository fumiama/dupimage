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
	d := flag.Bool("D", false, "enable debug-level log output")
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
	isdebu := *d
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
	duplis := make(map[uint]uint, len(chklst))
	sameset := make([][]uint, 0)
	for i := 0; i < len(chklst); i++ {
		fmt.Print("compare: ", i, " / ", len(chklst), "\r")
		if isdebu {
			fmt.Print("\n")
		}
		x, ok := duplis[uint(i)]
		if ok {
			if isdebu {
				fmt.Println(chklst[i].name, "already has index", x, "skip")
			}
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
				x, ok := duplis[uint(j)]
				if isdebu {
					fmt.Println("adding image index", j, "into same set", i)
				}
				if ok { // 该图片已被归入其他组
					hasfound := false
					if isdebu {
						fmt.Println("this image has been added into set", x)
					}
				LOP:
					for k, set := range sameset {
						for _, item := range set {
							if x == item { // 是该图片所属的组
								if isfirst { // 首次归类, 直接添加新组
									sameset[k] = append(sameset[k], uint(i))
									duplis[uint(i)] = uint(i)
									isfirst = false
									hasfound = true
									if isdebu {
										fmt.Println("first time appears, add directly:", sameset[k])
									}
								} else {
								INNERLOP:
									for l, set := range sameset {
										for _, item := range set {
											if item == uint(i) { // 找到旧组
												if l == k {
													if isdebu {
														fmt.Println("set", i, "and", x, "already in the same set")
													}
													hasfound = true
													break INNERLOP
												}
												if isdebu {
													fmt.Println("merge old set", set, "into", sameset[k])
												}
												sameset[k] = append(sameset[k], set...)         // 合并
												sameset = append(sameset[:l], sameset[l+1:]...) // 删除
												hasfound = true
												break INNERLOP
											}
										}
									}
								}
								break LOP
							}
						}
					}
					if !hasfound {
						fmt.Println("sameset:", sameset)
						panic("index " + strconv.Itoa(j) + ", file " + chklst[j].name + " has been marked as set " + strconv.FormatUint(uint64(x), 10) + " but cannot be found in sameset")
					}
				} else if isfirst { // 自立新组
					sameset = append(sameset, []uint{uint(i)})
					duplis[uint(i)] = uint(i)
					isfirst = false
					if isdebu {
						fmt.Println("new set:", i)
					}
				}
				duplis[uint(j)] = uint(i)
				if isdebu {
					fmt.Print("\n")
				}
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
			fmt.Println("sameset:", sameset)
			panic("cannot find index " + strconv.FormatUint(uint64(i), 10) + " in sameset")
		}
		for k, v := range duplis {
			i := setindex(v)
			dupset[i] = append(dupset[i], chklst[k].name)
		}
		j := *s
		for _, lst := range dupset {
			if len(lst) > 0 {
				j++
				newdir := strconv.FormatUint(uint64(j), 10)
				fmt.Println("["+newdir+"] duplicate:", lst)
				if action {
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
