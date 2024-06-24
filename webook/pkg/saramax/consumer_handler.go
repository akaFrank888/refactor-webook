package saramax

import (
	"encoding/json"
	"github.com/IBM/sarama"
	"refactor-webook/webook/pkg/logger"
)

type Handler[T any] struct {
	// note 执行在ConsumeClaim中的业务处理逻辑
	fn func(msg *sarama.ConsumerMessage, event T) error
	l  logger.LoggerV1
}

func NewHandler[T any](fn func(msg *sarama.ConsumerMessage, event T) error, l logger.LoggerV1) *Handler[T] {
	return &Handler[T]{fn: fn, l: l}
}

func (h *Handler[T]) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h *Handler[T]) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h *Handler[T]) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgs := claim.Messages()
	for msg := range msgs {
		// note 先反序列化成readEvent，再执行业务处理逻辑
		var t T
		err := json.Unmarshal(msg.Value, &t)
		if err != nil {
			// note 此处也可以引入 重试 逻辑
			// note 若反序列化失败，不能直接返回，要继续处理下去（结束后再查看日志单独消费）
			h.l.Error("反序列化失败",
				logger.String("topic", msg.Topic),
				logger.Int32("partition", msg.Partition),
				logger.Int64("offset", msg.Offset),
				logger.Error(err),
				// 这里也可以考虑打印 msg.Value，但是有些时候 msg 本身也包含敏感数据
			)
		}

		// 执行业务逻辑
		err = h.fn(msg, t)
		if err != nil {
			h.l.Error("处理消息失败",
				logger.String("topic", msg.Topic),
				logger.Int32("partition", msg.Partition),
				logger.Int64("offset", msg.Offset),
				logger.Error(err))
		}
		session.MarkMessage(msg, "")
	}
	return nil
}
