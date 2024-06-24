package channel

import (
	"testing"
	"time"
)

func TestChannel(t *testing.T) {
	ch := make(chan int, 1)
	ch <- 1
	val, ok := <-ch
	t.Log("val和ok", val, ok)
	close(ch)
}

func TestChannelLoop(t *testing.T) {
	ch := make(chan int)
	go func() {
		for i := 0; i < 3; i++ {
			ch <- i
			time.Sleep(time.Second)
		}
		close(ch)
	}()

	for val := range ch {
		t.Log(val)
	}

	t.Log("ch还有数据么？", <-ch)
	t.Log("结束")
}

func TestChannelSelect(t *testing.T) {
	ch1 := make(chan int)
	ch2 := make(chan int)

	go func() {
		ch1 <- 1
	}()
	go func() {
		ch2 <- 2
	}()

	select {
	case val := <-ch1:
		t.Log("进来了ch1", val)
	case val := <-ch2:
		t.Log("进来了ch2", val)
	}

}
