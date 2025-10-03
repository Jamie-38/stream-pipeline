package scheduler

import (
	"context"
	"fmt"
	"log"

	"github.com/Jamie-38/stream-pipeline/internal/types"
)

func Control_scheduler(ctx context.Context, controlCh <-chan types.IRCCommand, writerCh chan<- string) {
	for {
		select {
		case cmd := <-controlCh:
			switch cmd.Op {
			case "JOIN":
				writerCh <- fmt.Sprintf("JOIN %s\r\n", cmd.Channel)
			case "PART":
				writerCh <- fmt.Sprintf("PART %s\r\n", cmd.Channel)
			default:
				log.Printf("unknown IRC command: %+v", cmd)
			}
		case <-ctx.Done():
			return
		}
	}
}
