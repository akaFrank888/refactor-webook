package concurrent

import (
	"fmt"
	"sync"    // 导入sync包，提供同步原语如Mutex, RWMutex, WaitGroup等
	"testing" // 测试框架，用于定义和执行测试用例
	"time"    // 时间相关的函数，用于模拟延迟
)

// TestMutex函数展示了如何使用sync.Mutex来保护共享资源count免受并发写入的竞争
func TestMutex(t *testing.T) {
	var mu sync.Mutex // 声明一个互斥锁
	var count int     // 声明一个共享资源count

	// increment函数模拟一个原子性的递增操作
	increment := func() {
		mu.Lock()         // 在访问共享资源之前加锁
		defer mu.Unlock() // 访问完成后解锁，确保锁的正确释放
		count++           // 更新共享资源
		fmt.Println("Incremented count to", count)
		time.Sleep(100 * time.Millisecond) // 模拟一些耗时操作
	}

	// t.Run用于执行一个子测试
	t.Run("ConcurrentIncrement", func(t *testing.T) {
		var wg sync.WaitGroup // 声明一个WaitGroup来等待所有goroutine完成
		for i := 0; i < 10; i++ {
			wg.Add(1)      // 为即将启动的goroutine增加计数
			go increment() // 启动goroutine执行increment操作
		}
		wg.Wait()        // 等待所有goroutine完成
		if count != 10 { // 验证count的最终值是否符合预期
			t.Errorf("Expected count to be 10, got %d", count)
		}
	})
}

// TestDataRace函数展示了如何使用sync.RWMutex处理读写操作，防止数据竞争
func TestDataRace(t *testing.T) {
	// Data结构体包含一个共享资源value和一个读写锁rwmu
	type Data struct {
		value int
		rwmu  sync.RWMutex
	}

	// read函数模拟读取操作，使用读锁保护共享资源
	read := func(d *Data) {
		d.rwmu.RLock()         // 读取前加读锁
		defer d.rwmu.RUnlock() // 读取后解锁
		fmt.Printf("Reading value: %d\n", d.value)
		time.Sleep(10 * time.Millisecond) // 模拟读取耗时
	}

	// write函数模拟写入操作，使用写锁确保写操作的原子性
	write := func(d *Data, newValue int) {
		d.rwmu.Lock()         // 写入前加写锁，阻止其他读写操作
		defer d.rwmu.Unlock() // 写入后解锁
		d.value = newValue    // 更新共享资源
		fmt.Printf("Set value to: %d\n", newValue)
	}

	// t.Run执行另一个子测试
	t.Run("ReadWriteOperations", func(t *testing.T) {
		data := &Data{} // 创建一个Data实例

		// 模拟并发读取操作和一个写入操作
		var wg sync.WaitGroup
		for i := 0; i < 9; i++ {
			wg.Add(1)
			go read(data) // 并发读取
		}
		wg.Add(1)
		go write(data, 42) // 单独的写入操作

		wg.Wait()             // 确保所有goroutine执行完毕
		if data.value != 42 { // 验证写入操作是否成功
			t.Errorf("Expected value to be 42, got %d", data.value)
		}
	})
}

// 注意：这里的打印语句和time.Sleep是为了模拟并发操作的环境，真实测试中应根据具体情况设计更合理的测试逻辑。
