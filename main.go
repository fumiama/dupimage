package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"runtime"
	"strconv"
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
	flag.Parse()
	throttle := *tht
	if throttle > 64 {
		panic("invalid throttle")
	}
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
		go func(from, to int) {
			for i := from; i < to; i++ {
				img := imgs[i]
				if !img.IsDir() {
					f, err := os.Open(img.Name())
					if err != nil {
						panic(err)
					}
					im, _, err := image.Decode(f)
					if err != nil {
						panic(err)
					}
					dh, err := goimagehash.DifferenceHash(im)
					if err != nil {
						panic(err)
					}
					mu.Lock()
					chklst = append(chklst, imagecheck{
						name: img.Name(),
						dh:   dh,
					})
					fmt.Print("scan: ", len(chklst), " / ", len(imgs), "\r")
					mu.Unlock()
					_ = f.Close()
				}
			}
			wg.Done()
		}(from, to)
	}
	wg.Wait()
	fmt.Println("read file success, comparing...")
	dups := make([][]*imagecheck, len(chklst))
	wg.Add(len(chklst))
	for i := 0; i < len(chklst); i++ {
		go func(i int) {
			for j := len(chklst) - 1; j > i; j-- {
				dis, err := chklst[i].dh.Distance(chklst[j].dh)
				if err != nil {
					panic(err)
				}
				if uint(dis) < throttle {
					mu.Lock()
					dups[i] = append(dups[i], &chklst[j])
					mu.Unlock()
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	fmt.Println("compare file success")
	hasfound := false
	for i, lst := range dups {
		if len(lst) > 0 {
			dups[i] = append(dups[i], &chklst[i])
			hasfound = true
		}
	}
	if hasfound {
		j := 0
		for _, lst := range dups {
			if len(lst) > 0 {
				j++
				fmt.Println("[", j, "] duplicate: ", lst)
				if action {
					newdir := strconv.Itoa(j)
					err = os.MkdirAll(newdir, 0755)
					if err != nil {
						panic(err)
					}
					for _, i := range lst {
						err = os.Rename(i.name, newdir+"/"+i.name)
						if err != nil {
							panic(err)
						}
					}
				}
			}
		}
	}
}
