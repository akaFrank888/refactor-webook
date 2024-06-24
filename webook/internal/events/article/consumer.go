package article

import (
	"context"
	"github.com/IBM/sarama"
	"refactor-webook/webook/internal/repository"
	"refactor-webook/webook/pkg/logger"
	"refactor-webook/webook/pkg/saramax"
	"time"
)

type InteractiveReadEventConsumer struct {
	repo   repository.InteractiveRepository
	client sarama.Client
	l      logger.LoggerV1
}

func NewInteractiveReadEventConsumer(repo repository.InteractiveRepository, client sarama.Client, l logger.LoggerV1) *InteractiveReadEventConsumer {
	return &InteractiveReadEventConsumer{repo: repo, client: client, l: l}
}

// Start note 启动 consumer 的方法，实现了自定义接口
func (i *InteractiveReadEventConsumer) Start() error {
	cg, err := sarama.NewConsumerGroupFromClient("interactive", i.client)
	if err != nil {
		return err
	}

	go func() {
		er := cg.Consume(context.Background(), []string{TopicReadEvent}, saramax.NewHandler[ReadEvent](i.Consume, i.l))
		if er != nil {
			i.l.Error("消费消息出现错误", logger.Error(er))
		}
	}()

	return err
}

func (r *InteractiveReadEventConsumer) StartBatch() error {
	cg, err := sarama.NewConsumerGroupFromClient("interactive",
		r.client)
	if err != nil {
		return err
	}
	go func() {
		err := cg.Consume(context.Background(),
			[]string{TopicReadEvent},
			saramax.NewBatchHandler[ReadEvent](r.l, r.BatchConsume))
		if err != nil {
			r.l.Error("退出了消费循环异常", logger.Error(err))
		}
	}()
	return err
}

// Consume 对应 saramax.Handler中的fn方法
func (i *InteractiveReadEventConsumer) Consume(msg *sarama.ConsumerMessage, event ReadEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return i.repo.IncrReadCnt(ctx, "article", event.Aid)
}

func (r *InteractiveReadEventConsumer) BatchConsume(msgs []*sarama.ConsumerMessage,
	evts []ReadEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	bizs := make([]string, 0, len(msgs))
	ids := make([]int64, 0, len(msgs))
	for _, evt := range evts {
		bizs = append(bizs, "article")
		ids = append(ids, evt.Uid)
	}
	return r.repo.BatchIncrReadCnt(ctx, bizs, ids)
}
