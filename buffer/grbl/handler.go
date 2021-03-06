package grbl

import (
	"errors"
	"strings"

	"github.com/mastercactapus/yaspjs/buffer"
)

const grblMax = 125

type Grbl struct {
	q *buffer.Queue

	feedHold bool

	version    string
	lastStatus string
}

var _ buffer.Handler = &Grbl{}

func filterJog(cmd string) bool { return !strings.HasPrefix(cmd, "$J=") }

func NewHandler() buffer.Handler {
	return &Grbl{
		// all commands will be at least 1 byte + 1 newline, 64*2 = 128 which is already larger than Grbl's 127-byte buffer.
		q: buffer.NewQueue(),
	}
}

func (g *Grbl) Buffer() []interface{} { return g.q.Buffer() }

func (g *Grbl) PollCommand() string { return "?" }
func (g *Grbl) CheckBuffer(data string) bool {
	return g.q.ByteLen()+len(data) <= grblMax
}
func (g *Grbl) IsPaused() bool { return g.feedHold }

func (g *Grbl) HandleMeta(cmd string) string {
	switch cmd {
	case "*init*":
		return g.version
	case "*status*":
		return g.lastStatus
	}

	return ""
}

func (g *Grbl) HandleInput(input buffer.QueueItem) []buffer.CommandResponse {
	if g.feedHold && filterJog(input.Data) {
		return []buffer.CommandResponse{{QueueItem: input, Err: errors.New("jog unavailable")}}
	}

	if len(input.Data) == 1 {
		// no control characters expect a response
		return []buffer.CommandResponse{{QueueItem: input, Done: true}}
	}

	g.q.Push(input)
	if g.q.ByteLen() > grblMax {
		panic("overflow")
	}
	return nil
}

func (Grbl) FlowConfig() buffer.FlowConfig {
	return buffer.FlowConfig{
		InputSplitFunc:    ScanInput,
		SplitControlChars: buffer.SplitStaticControlChars("\x18?~!\x84\x85\x90\x91\x92\x93\x94\x95\x96\x97\x99\x9a\x9b\x9c\x9c\x9d\x9e\xa0\xa1"),
		IsControl:         func(cmd string) bool { return strings.HasPrefix(cmd, "$J=") },
		IsMeta: func(cmd string) bool {
			return strings.HasPrefix(cmd, "*") || cmd == "%"
		},
		IsBufferReset: func(cmd string) bool { return cmd == "\x18" || cmd == "%" },
		IsPartialBufferReset: func(cmd string) func(cmd string) bool {
			switch cmd {
			case "!", "\x84", "\x85":
			default:
				return nil
			}

			return filterJog
		},
	}
}

func (g *Grbl) HandleResponse(data string) []buffer.CommandResponse {
	if data == "ok" {
		return []buffer.CommandResponse{{
			QueueItem: g.q.Shift().(buffer.QueueItem),
			Done:      true,
		}}
	}
	if strings.HasPrefix(data, "error:") {
		return []buffer.CommandResponse{{
			QueueItem: g.q.Shift().(buffer.QueueItem),
			Err:       errors.New(data),
		}}
	}
	if strings.HasPrefix(data, "Grbl") {
		g.version = data
		items := g.q.Reset()
		resp := make([]buffer.CommandResponse, len(items))
		for i, item := range items {
			resp[i].QueueItem = item.(buffer.QueueItem)
			resp[i].Err = errors.New("reset")
		}
	}
	if strings.HasPrefix(data, "<") {
		g.lastStatus = data
	}
	return nil
}
