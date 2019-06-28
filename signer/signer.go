package main

import (
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	in := make(chan interface{})

	for _, job := range jobs {
		wg.Add(1)
		out := make(chan interface{})
		go jobGo(job, in, out, wg)
		in = out

	}
	wg.Wait()
}

func jobGo(job job, in chan interface{}, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(out)
	job(in, out)
}

func SingleHash(in, out chan interface{}) {
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	for i := range in {
		data, ok := i.(int)
		if !ok {
			fmt.Print("Error cannot convert to int")
		}

		wg.Add(1)
		go SH(strconv.Itoa(data), out, mu, wg)
		runtime.Gosched()
	}

	wg.Wait()

}

func SH(data string, out chan interface{}, mu *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	mu.Lock()
	md5 := DataSignerMd5(data)
	mu.Unlock()

	incrc := make(chan string)
	outcrc := make(chan string)
	incrcmd5 := make(chan string)
	outcrcmd5 := make(chan string)
	defer close(incrc)
	defer close(outcrc)
	defer close(incrcmd5)
	defer close(outcrcmd5)

	go crcGo(incrc, outcrc)
	go crcGo(incrcmd5, outcrcmd5)

	incrc <- data
	incrcmd5 <- md5
	crc := <-outcrc
	crcmd5 := <-outcrcmd5

	out <- (crc + "~" + crcmd5)
}

func crcGo(in, out chan string) {
	data := <-in
	out <- DataSignerCrc32(data)
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}

	for data := range in {
		wg.Add(1)
		go MH(data.(string), out, wg)
		runtime.Gosched()

	}
	wg.Wait()
}

type MHstruct struct {
	index int
	res   string
}

func MH(data string, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	num := 6

	outCrc := make(chan MHstruct)
	defer close(outCrc)

	for i := 0; i < num; i++ {
		go crcMHgo(data, i, outCrc)

		runtime.Gosched()
	}

	arrRes := make([]string, num)
	for i := 0; i < num; i++ {
		crcOutput := <-outCrc
		arrRes[crcOutput.index] = crcOutput.res
	}

	res := strings.Join(arrRes, "")
	out <- res
	fmt.Print("data = " + data + " MH = ")
	fmt.Print(res + "\n")
}

func crcMHgo(data string, i int, out chan MHstruct) {
	var res MHstruct
	res = MHstruct{
		index: i,
		res:   DataSignerCrc32(strconv.Itoa(i) + data),
	}
	fmt.Print("data = " + data + " crc(th+data) = ")
	fmt.Print(strconv.Itoa(i) + " " + res.res + "\n")

	out <- res
}

func CombineResults(in, out chan interface{}) {
	all := make([]string, 0, 10)
	for data := range in {
		all = append(all, data.(string))
	}

	sort.Strings(all)

	combined := strings.Join(all, "_")
	out <- combined
}

func main() {
	ExecutePipeline()
}
