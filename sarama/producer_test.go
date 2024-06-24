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
	cfg.Producer.Return.Successes = true
	// note 可以指定partitioner
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
	cfg.Producer.Return.Successes = true
	cfg.Producer.Return.Errors = true
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
