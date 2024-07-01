package concurrent

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// 定义一个工作项类型
type WorkItem int

// 生产者函数
func producer(ch chan<- WorkItem, wg *sync.WaitGroup) {
	defer wg.Done() // 通知等待组，此goroutine完成

	for i := 1; i <= 5; i++ { // 生产5个工作项
		ch <- WorkItem(i) // 将工作项发送到channel
		fmt.Printf("Producer sent: %v\n", i)
		time.Sleep(time.Second) // 模拟生产间隔
	}
	close(ch) // 生产完毕后关闭channel
}

// 消费者函数
func consumer(ch <-chan WorkItem, wg *sync.WaitGroup) {
	defer wg.Done()

	for item := range ch { // 阻塞等待接收channel中的数据
		fmt.Printf("Consumer received: %v\n", item)
		time.Sleep(time.Second * 2) // 模拟消费间隔
	}
}

func Test_producer_consumer(t *testing.T) {
	var wg sync.WaitGroup // 使用WaitGroup来等待所有goroutine完成

	// 创建一个channel用于goroutine间通信
	workChan := make(chan WorkItem)

	// 启动生产者goroutine
	wg.Add(1)
	go producer(workChan, &wg)

	// 启动消费者goroutine
	wg.Add(1)
	go consumer(workChan, &wg)

	// 等待所有goroutine完成
	wg.Wait()
	fmt.Println("All tasks completed.")
}
