package scripts

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"
)

func TestWrong(t *testing.T) {
	var s1 = []Persion{
		{
			Name: "a",
			Age:  1,
		},
		{
			Name: "b",
			Age:  2,
		},
		{
			Name: "c",
			Age:  3,
		},
	}
	var s2 = []Persion{
		{
			Name: "d",
			Age:  11,
		},
		{
			Name: "e",
			Age:  12,
		},
	}

	var strName = "a"
	var oPersion *Persion
	var index int
	for i, tmp := range s1 {
		if tmp.Name == strName {
			oPersion = &s1[i]
			index = i
			break
		}
	}
	for _, oPersion2 := range s2 {
		s1 = append(s1, oPersion2)
	}
	oPersion.Age = CulAget()
	//s1[index]
	fmt.Println(oPersion)
	fmt.Println(s1)
	fmt.Println(index)

}

type Persion struct {
	Name string
	Age  int
}

func CulAget() int {
	return 19
}

func TestQuick(t *testing.T) {
	res := QuickSort([]int{1, 2, 5, 6, 7})
	//res = arr([]int{1, 2, 5, 6, 7})
	t.Log(res)
}

func arr(arg []int) []int {

	if len(arg) <= 1 {
		return arg
	}
	var left []int
	var right []int
	pivot := arg[0]
	for _, item := range arg[1:] {
		if item <= pivot {
			left = append(left, item)
		} else {
			right = append(right, item)
		}
	}
	left = arr(left)
	right = arr(right)

	return append(append(left, pivot), right...)
}
func QuickSort(arr []int) []int {
	if len(arr) <= 1 {
		return arr
	}
	pivot := arr[0]
	var left, right []int
	for _, v := range arr[1:] {
		if v <= pivot {
			left = append(left, v)
		} else {
			right = append(right, v)
		}
	}
	left = QuickSort(left)
	right = QuickSort(right)
	return append(append(left, pivot), right...)
}

type LetterFreq map[rune]int

func CountLetters(str []string, concurrency uint) LetterFreq {
	res := make(map[rune]int) //

	ch := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, item := range str {
		wg.Add(1)
		ch <- struct{}{}
		go func(item string) {
			defer wg.Done()
			time.Sleep(time.Second * 2)
			local := make(LetterFreq)
			for _, v := range item {
				local[v]++
			}
			mu.Lock()
			for _, v := range item {
				res[v]++
			}
			mu.Unlock()

			<-ch
		}(item)
		fmt.Println(ch, len(ch))
	}
	wg.Wait()

	return LetterFreq(res)
}

func TestCounter(t *testing.T) {
	value := []string{"abc", "ab", "b", "c", "e", "f"}
	res := CountLetters(value, 2)
	t.Log(res)
}

func CountLetters1(str []string, concurrency uint) LetterFreq {
	var wg sync.WaitGroup
	var mu sync.Mutex
	res := make(LetterFreq)

	// 创建一个限制最大并发量的通道
	semaphore := make(chan struct{}, concurrency)

	for _, item := range str {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			// 获取信号量
			semaphore <- struct{}{}
			// 当前goroutine结束时释放信号量
			defer func() { <-semaphore }()
			localRes := make(LetterFreq)
			for _, v := range s {
				localRes[v]++
			}

			// 合并结果到共享map
			mu.Lock()
			for k, v := range localRes {
				res[k] += v
			}
			mu.Unlock()
		}(item)
	}

	wg.Wait()
	return res
}

func TestSlice(t *testing.T) {
	x := []int{1, 2, 3}
	y := x[:2]
	t.Log(y) //[1 2]
	y = append(y, 40)
	y = append(y, 50) //扩容了指向了不同空间
	t.Log(x)          //[1 2 40]
	y[0] = 100
	t.Log(x) // [1 2 40]
	t.Log(y) // [100 2 40 50]

}

// 1. 项目经验（字节对齐）
// 2. 代码题 count letter
// 3. map make其他的声明方式（2相关）
// 4. 代码的最大并发，channel相关，没仔细看（2相关）
// 5. slice append后的结果
// 6. slice 性能优化，减少扩容次数
// 7. goroutine 死锁分析，解决，加channel缓冲
// 8. ticker 陷阱，select 随机选取问题，default解决

type Obj struct {
	id int
}

func createObjs(n int) []*Obj {

	objs := make([]Obj, n)
	refs := make([]*Obj, 0, n)

	//refs := make([]*Obj) 问题扩容
	for i := 0; i < n; i++ {
		//obj := Obj{id: i}
		//		refs[i] = &obj
		//refs = append(refs, &obj)
		refs = append(refs, &objs[i])
	}
	return refs
}
func createObjsOne(n int) []*Obj {
	// 一次性申请内存空间来存放所有Obj。
	objs := make([]Obj, n)
	refs := make([]*Obj, n)

	for i := range objs {
		objs[i].id = i
		refs[i] = &objs[i]
	}

	return refs
}

var (
	execTime time.Duration = time.Second
	wg       sync.WaitGroup
)

// 考察select 处理后无法退出 channel 阻塞的坑
func finishReq(timeout time.Duration) int {
	ch := make(chan int) //增加容量
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(execTime)
		ch <- 42
	}()
	select { //
	case result := <-ch:
		return result
	case <-time.After(timeout):
		log.Print("timeout")
		return -1
	}
}

func TestFinsh(t *testing.T) {
	timeout := 50 * time.Millisecond //时间过短
	t.Logf("finishReq: %d", finishReq(timeout))
	wg.Wait()
}

func stopCh() {
	stopCh := make(chan struct{})
	ticker := time.NewTicker(100 * time.Millisecond)

	go func() {
		time.Sleep(2 * time.Second)
		close(stopCh)
	}()
	for {
		select {
		case <-stopCh:
			log.Print("exit-1")
			return
		default:
		}
		time.Sleep(time.Second)
		log.Print("woking hard")

		select {
		case <-ticker.C:
			log.Print("next")
		case <-stopCh:
			log.Print("exit-2")
			return
		}
	}

}

func TestStopC(t *testing.T) {
	stopCh()
}
