package httpapi

import (
	"github.com/Jamie-38/stream-pipeline/internal/types"
)

type APIController struct {
	ControlCh chan types.IRCCommand
}
