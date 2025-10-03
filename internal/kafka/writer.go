package kafka

import (
	kafkago "github.com/segmentio/kafka-go"
)

func New_Writer() *kafkago.Writer {
	w := &kafkago.Writer{
		Addr:     kafkago.TCP("localhost:9094"),
		Topic:    "test-topic",
		Balancer: &kafkago.LeastBytes{},
	}
	return w
}
