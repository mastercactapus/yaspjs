package buffer

import (
	"io"
	"time"
)

type Config struct {
	io.ReadWriteCloser
	Handler
	OnRead   func(string)
	OnUpdate (func(CommandResponse))

	PollInterval time.Duration
}
