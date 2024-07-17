package sarama

import (
	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"testing"
)

var addr = []string{"localhost:9094"}

// note 在 webook 目录下执行 `kafka-console-consumer -topic=test_topic -brokers=localhost:9094`查看是否发送了消息
func TestSyncProducer(t *testing.T) {
	cfg := sarama.NewConfig()
	// note 使生产者在成功发送消息后返回成功状态。这有助于测试函数获取到消息是否成功发送的信息
	cfg.Producer.Return.Successes = true
	// note 可以指定 partitioner
	cfg.Producer.Partitioner = sarama.NewRandomPartitioner
	//cfg.Producer.Partitioner = sarama.NewRandomPartitioner
	//cfg.Producer.Partitioner = sarama.NewHashPartitioner
	//cfg.Producer.Partitioner = sarama.NewManualPartitioner
	//cfg.Producer.Partitioner = sarama.NewConsistentCRCHashPartitioner
	//cfg.Producer.Partitioner = sarama.NewCustomPartitioner()

	producer, err := sarama.NewSyncProducer(addr, cfg)

	assert.NoError(t, err)
	err = producer.SendMessages([]*sarama.ProducerMessage{
		{
			Topic: "test_topic",
			Key:   sarama.StringEncoder("这条msg对应的key"),
			Value: sarama.StringEncoder("这是一条test_value"),
			Headers: []sarama.RecordHeader{
				{
					Key:   []byte("test_key"),
					Value: []byte("这是一条test_value"),
				},
			},
			Metadata: "test_metadata",
		},
	})
	assert.NoError(t, err)
}

func TestAsyncProducer(t *testing.T) {
	cfg := sarama.NewConfig()
	// 为了后面能够拿到发送结果
	cfg.Producer.Return.Successes = true
	cfg.Producer.Return.Errors = true

	// note 从上到下，性能变差，但是数据可靠性上升
	cfg.Producer.RequiredAcks = sarama.NoResponse   // 客户端发一次，不需要服务端的确认
	cfg.Producer.RequiredAcks = sarama.WaitForLocal // 客户端发送，并且需要服务端写入到主分区
	cfg.Producer.RequiredAcks = sarama.WaitForAll   // 客户端发送，并且需要服务端同步到所有的 ISR (In Sync Replicas，就是跟上了节奏的从分区)

	producer, err := sarama.NewAsyncProducer(addr, cfg)
	assert.NoError(t, err)

	msgCh := producer.Input()
	msgCh <- &sarama.ProducerMessage{
		Topic: "test_topic",
		Value: sarama.StringEncoder("这是一条test_value"),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("test_key"),
				Value: []byte("这是一条test_value"),
			},
		},
		Metadata: "test_metadata",
	}

	select {
	case err := <-producer.Errors():
		t.Log("发送失败,", err.Err, err.Msg)
	case msg := <-producer.Successes():
		t.Log("发送成功", string(msg.Value.(sarama.StringEncoder)))
	}
}
