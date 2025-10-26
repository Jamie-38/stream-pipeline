package channelrecord

import (
	"context"

	"github.com/Jamie-38/stream-pipeline/internal/types"
)

func Run(ctx context.Context, channels types.Channels, controlCh <-chan types.IRCCommand, irceventCh chan<- types.IRCCommand) {

}
