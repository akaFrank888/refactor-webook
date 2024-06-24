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

	// 10s后context过期，进而关闭consumer
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

// 若是人为关闭Consumer，则不会触发Cleanup；若是context超时，则会执行Cleanup
func (c ConsumerHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	// 执行一些清理工作
	log.Println("这是Handler Cleanup。。。")
	return nil
}

// note 异步消费、批量提交  【消费一批，提交一批】
func (c ConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgs := claim.Messages()
	const batchSize = 10
	for {
		batch := make([]*sarama.ConsumerMessage, 0, batchSize) // 构建容量为10的切片
		var eg errgroup.Group
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		done := false

		for i := 0; i < batchSize; i++ {
			select {
			case <-ctx.Done():
				// 这个批次的ctx超时了
				done = true
			case msg, ok := <-msgs: // note 需要ok，因为channel有可能被关闭
				if !ok {
					// channel被关闭了
					cancel()
					return nil
				}
				batch = append(batch, msg)
				// note 异步消费（并发）
				eg.Go(func() error {
					// 并发处理
					log.Println("【并发】Consumer消费来自producer的消息：", string(msg.Value))
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

func (c ConsumerHandler) ConsumeClaimV1(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgs := claim.Messages()
	for msg := range msgs {
		log.Println("Consumer收到来自producer的消息：", string(msg.Value))
		// 提交
		session.MarkMessage(msg, "")
	}
	return nil
}
