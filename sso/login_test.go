package sso

import (
	"fmt"
	"sync"
	"testing"
)

func TestBizServer(t *testing.T) {

}

// 实现一个可以从多个goroutine安全递增的并发计数器。
type Counter struct {
	value int
}

func (c *Counter) increment() {
	c.value++
}

func add() {
	var wg sync.WaitGroup
	counter := Counter{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.increment()
		}()
	}
	wg.Wait()
	fmt.Println("Counter value:", counter.value)
}

// prompt提示词是:实现一个可以从多个goroutine安全递增的并发计数器。

// 关键词是,多个goroutine,并发, 安全递增,
// 代码实现了多个 Goroutine 并发执行 `c.value++` 的递增操作，并使用 `sync.WaitGroup` 控制主 Goroutine 等待所有内部 Goroutine 执行完成。然而，由于对结构体 `Counter` 的写操作没有得到保护或有序执行，无法实现并发安全。

// `Increment` 方法没有使用锁或其他同步机制来保护对 `value` 字段的访问，当多个 Goroutine 同时调用 `Increment` 方法时，会发生数据竞争。

// go实现并发安全常用的方式是：

// 1. 原子操作：使用 Go 语言提供的原子操作函数，例如 `atomic.AddInt64` 和 `atomic.LoadInt64`，来对 `value` 字段进行原子递增和读取操作，以确保并发安全。

// 2. 使用互斥锁（Mutex）：在 `Increment` 方法中使用互斥锁来保护对 `value` 字段的访问。在递增操作之前获取锁，
// 在递增操作完成后释放锁，以确保同一时间只有一个 Goroutine 对 `value` 字段进行写操作。

// 3. 使用通道（Channel）进行通信：通过创建一个带缓冲的通道，
// 可以将递增操作封装在一个匿名函数中，并将该函数作为 Goroutine 运行。
// 然后在主 Goroutine 中使用通道来接收完成信号，确保只有一个 Goroutine 在进行写操作。

// Go 的并发模型是基于（CSP），提倡通过通信来共享内存，而不是通过共享内存来实现通信。
// 它通过 Goroutine 和 Channel 实现，这种方式可以确保并发操作的安全性和正确性。
