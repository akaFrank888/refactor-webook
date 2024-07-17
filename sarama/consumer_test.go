package sarama

import (
	"context"
	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
	"log"
	"testing"
	"time"
)

func TestConsumer(t *testing.T) {
	cfg := sarama.NewConfig()
	group, err := sarama.NewConsumerGroup(addr, "demo", cfg)
	assert.NoError(t, err)

	// 10s后context过期 or cancel 使得关闭consumer，退出消费
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	start := time.Now()
	// note consumer 不关闭，则会一直阻塞在这里
	err = group.Consume(ctx, []string{"test_topic"}, ConsumerHandler{})
	assert.NoError(t, err)
	t.Log(time.Since(start))
}

type ConsumerHandler struct {
}

func (c ConsumerHandler) Setup(session sarama.ConsumerGroupSession) error {
	// 执行一些初始化的事情
	log.Println("这是Handler Setup。。。")
	return nil
}

// note 若是人为关闭Consumer，则不会触发Cleanup；若是context超时，则会执行Cleanup
func (c ConsumerHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	// 执行一些清理工作
	log.Println("这是Handler Cleanup。。。")
	return nil
}

// note 异步消费、批量提交（引入超时，避免一直阻塞在 <- msgs 中）
func (c ConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgs := claim.Messages()
	const batchSize = 10
	for {
		batch := make([]*sarama.ConsumerMessage, 0, batchSize) // 构建容量为10的切片
		var eg errgroup.Group                                  // 实现异步消费
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		done := false
		for i := 0; i < batchSize; i++ {
			select {
			case <-ctx.Done():
				// ctx超时了，避免一直组赛在 msg := <- msgs里
				// note 在case中使用break不起作用，所以引入 done 作为标记
				done = true
			case msg, ok := <-msgs: // note 需要ok，因为channel有可能被关闭
				if !ok {
					// channel被关闭了
					cancel()
					return nil
				}
				batch = append(batch, msg)
				// 异步消费
				eg.Go(func() error {
					// 执行消费业务
					log.Println("【并发】Consumer消费来自producer的消息：", string(msg.Value))
					// 模拟业务执行所需要的时间
					time.Sleep(time.Second * 3)
					return nil
				})
			}
			if done {
				break
			}
		}
		cancel()
		if err := eg.Wait(); err != nil {
			log.Println(err)
			continue
		}
		// note 批量提交
		for _, msg := range batch {
			session.MarkMessage(msg, "")
		}
	}
}

// 异步消费，批量提交
func (c ConsumerHandler) ConsumeClaimV3(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgCh := claim.Messages()
	for {
		const batchSize = 10
		batch := make([]*sarama.ConsumerMessage, 0, batchSize)
		eg := errgroup.Group{}
		for i := 0; i < batchSize; i++ {
			msg := <-msgCh
			batch = append(batch, msg)
			eg.Go(func() error {
				log.Println("实现异步消费")
				return nil
			})
		}

		err := eg.Wait()
		if err != nil {
			log.Println("异步消费出现了错误")
		}

		// 在这里实现 批量提交
		for _, msg := range batch {
			session.MarkMessage(msg, "")
		}
	}
}

// 批量消费，批量提交
func (c ConsumerHandler) ConsumeClaimV2(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgCh := claim.Messages()
	for {
		const batchSize = 10
		batch := make([]*sarama.ConsumerMessage, 0, batchSize)
		for i := 0; i < batchSize; i++ {
			msg := <-msgCh
			batch = append(batch, msg)
		}

		// 在这里实现 批量消费和批量提交
		log.Println(batch)
		for _, msg := range batch {
			session.MarkMessage(msg, "")
		}
	}
}

func (c ConsumerHandler) ConsumeClaimV1(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgs := claim.Messages()
	for msg := range msgs {
		log.Println("Consumer收到来自producer的消息：", string(msg.Value))
		// 提交
		session.MarkMessage(msg, "")
	}
	return nil
}
