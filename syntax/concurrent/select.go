package concurrent

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// 任务类型，包含优先级和具体操作
type Task struct {
	Priority int
	Action   func()
}

// 通道类型定义
type TaskChannel chan *Task

// 此代码模拟了一个简单的任务调度器：
// 其中高优先级任务（通过highPriorityTasks channel）可以中断正在进行的低优先级任务（通过lowPriorityTasks channel），
// 并且任何任务执行都受限于一个总体的超时限制。
func TestSelectComplexControlFlow(t *testing.T) {
	var wg sync.WaitGroup // 用于等待所有任务完成

	// 低优先级任务通道
	lowPriorityTasks := make(TaskChannel, 10)
	// 高优先级任务通道
	highPriorityTasks := make(TaskChannel, 10)

	// 超时定时器
	timeout := time.After(5 * time.Second)

	// 任务处理器
	go func() {
		defer wg.Done()

		for {
			select {
			case task, ok := <-lowPriorityTasks:
				if !ok { // 如果通道已关闭，退出循环
					return
				}
				if task.Priority == 7 { // 模拟优先级判断，这里只是一个示例
					fmt.Println("Interrupting low priority task for high priority")
					continue // 忽略当前低优先级任务
				}
				task.Action() // 执行任务
				fmt.Println("Low priority task done")

			case task, ok := <-highPriorityTasks:
				if !ok { // 如果通道已关闭，退出循环
					return
				}
				task.Action() // 立即执行高优先级任务
				fmt.Println("High priority task done")

			case <-timeout: // 超时处理
				fmt.Println("Timeout reached, stopping task execution")
				return // 超时后结束任务处理器
			}
		}
	}()

	// 添加一些示例任务
	wg.Add(1) // 为任务处理器增加一个等待计数
	lowPriorityTasks <- &Task{
		Priority: 3, Action: func() {
			time.Sleep(2 * time.Second)
			fmt.Println("Executing low priority task")
		},
	}

	highPriorityTasks <- &Task{
		Priority: 9, Action: func() {
			fmt.Println("Executing high priority task")
		},
	}

	// 关闭通道以结束处理器goroutine
	close(lowPriorityTasks)
	close(highPriorityTasks)

	wg.Wait() // 等待所有任务（包括任务处理器）完成
}
