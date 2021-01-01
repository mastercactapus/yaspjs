package buffer

import (
	"bufio"
	"strings"
)

type ControlCharSplitFunc func(input string) (ctrl []rune, data string)

type FlowConfig struct {
	// SplitControlChars splits any control characters that need to be stripped from
	// the input. These should be single-runes that do not need to end with a newline.
	//
	// For example: Grbl's `?`
	//
	// A stream of: `g0x?1\n` will have the `?` pulled and sent immediately
	// followed by `g0x1\n` being passed to the send buffer.
	//
	// Characters defined here will not be passed to `IsControl` and will instead be treated
	// as if it returned true. They are prioritized above Control lines.
	SplitControlChars ControlCharSplitFunc

	// WrapInput is used to wrap a raw command for sending. The default is to append a newline (`\n`).
	WrapInput func(string) string

	// SendSplitFunc can be specified to override using bufio.ScanLines.
	SendSplitFunc bufio.SplitFunc

	// RecvSplitFunc can be specified to override using bufio.ScanLines.
	RecvSplitFunc bufio.SplitFunc

	// IsControl should return true if a command should be sent before
	// any other pending data. Control commands are sent even in an error state.
	IsControl func(cmd string) bool

	// IsMeta should return true if a command is intended for the buffer
	// handler and not the actual serial port. (e.g. `*init*`) It will
	// be passed to `HandleMeta` instead of written to the port. Meta commands
	// are prioritized and do not count against any buffer limits.
	IsMeta func(cmd string) bool

	// IsBufferReset should return true if the command is expected to reset the data buffer.
	IsBufferReset func(cmd string) bool

	// IsPartialBufferReset should return a filter func if the command is expected to partially
	// reset the buffer. Values returned true will be kept.
	IsPartialBufferReset func(cmd string) func(cmd string) bool
}

func (cfg FlowConfig) WithDefaults() FlowConfig {
	if cfg.SplitControlChars == nil {
		cfg.SplitControlChars = func(input string) (ctrl []rune, data string) {
			return nil, input
		}
	}
	if cfg.RecvSplitFunc == nil {
		cfg.RecvSplitFunc = bufio.ScanLines
	}
	if cfg.SendSplitFunc == nil {
		cfg.SendSplitFunc = bufio.ScanLines
	}
	if cfg.IsControl == nil {
		cfg.IsControl = func(string) bool { return false }
	}
	if cfg.IsBufferReset == nil {
		cfg.IsBufferReset = func(string) bool { return false }
	}
	if cfg.IsPartialBufferReset == nil {
		cfg.IsPartialBufferReset = func(string) func(string) bool { return func(string) bool { return true } }
	}
	if cfg.WrapInput == nil {
		cfg.WrapInput = func(data string) string { return data + "\n" }
	}

	return cfg
}

// SplitStaticControlChars returns a ControlCharSplitFunc that will remove any instance of the provided characters.
func SplitStaticControlChars(chars string) ControlCharSplitFunc {
	return func(input string) (ctrl []rune, data string) {
		if !strings.ContainsAny(input, chars) {
			return nil, input
		}

		var buf strings.Builder
		buf.Grow(len(input))

		for _, c := range input {
			if strings.ContainsRune(chars, c) {
				ctrl = append(ctrl, c)
				continue
			}
			buf.WriteRune(c)
		}

		return ctrl, buf.String()
	}
}
