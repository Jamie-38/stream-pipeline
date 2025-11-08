package scheduler

import (
	"context"
	"fmt"

	"github.com/Jamie-38/stream-pipeline/internal/observe"
	"github.com/Jamie-38/stream-pipeline/internal/types"
)

func Control_scheduler(ctx context.Context, controlCh <-chan types.IRCCommand, writerCh chan<- string) {
	lg := observe.C("scheduler")
	for {
		select {
		case cmd := <-controlCh:
			switch cmd.Op {
			case "JOIN":
				lg.Debug("forwarded JOIN", "channel", cmd.Channel)
				writerCh <- fmt.Sprintf("JOIN %s\r\n", cmd.Channel)
			case "PART":
				lg.Debug("forwarded PART", "channel", cmd.Channel)
				writerCh <- fmt.Sprintf("PART %s\r\n", cmd.Channel)
			default:
				lg.Warn("unknown IRC command", "op", cmd.Op, "channel", cmd.Channel)
			}
		case <-ctx.Done():
			lg.Info("stopping", "reason", "context_canceled")
			return
		}
	}
}
