package server

import (
	"github.com/mastercactapus/yaspjs/buffer"
)

type Port struct {
	*buffer.Buffer

	name       string
	bufferType string
	baudRate   int
	primary    bool
}
