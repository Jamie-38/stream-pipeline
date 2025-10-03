package kafka

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

func KafkaProducer(ctx context.Context, writer *kafka.Writer, readerCh <-chan string) {
	for {
		select {
		case line := <-readerCh:
			err := writer.WriteMessages(ctx, kafka.Message{
				Key:   []byte("raw"),
				Value: []byte(line),
			})
			if err != nil {
				log.Println("kafka write error:", err)
				// optionally continue / backoff depending on policy
			}

		case <-ctx.Done():
			return
		}
	}
}
