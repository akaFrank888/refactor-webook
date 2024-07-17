package events

type Consumer interface {
	// Start 为了要在 main() 里启动消费者
	Start() error
	StartBatch() error
}
