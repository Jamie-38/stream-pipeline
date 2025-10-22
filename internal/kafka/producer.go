package kafka

import (
	"context"
	"log"

	kafkago "github.com/segmentio/kafka-go"

	ircevents "github.com/Jamie-38/stream-pipeline/internal/irc_events"
)

func KafkaProducer(ctx context.Context, writer *kafkago.Writer, parseCh <-chan ircevents.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-parseCh:
			value, err := evt.Marshal()
			if err != nil {
				log.Println("marshal error:", err)
				continue
			}
			msg := kafkago.Message{
				Key:   []byte(evt.Key()),
				Value: value,
				// optional: Headers: []kafkago.Header{{Key: "event-kind", Value: []byte(evt.Kind())}},
			}
			if err := writer.WriteMessages(ctx, msg); err != nil {
				log.Println("kafka write error:", err)
			}
		}
	}
}
