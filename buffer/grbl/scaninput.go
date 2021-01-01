package grbl

import (
	"bufio"
	"bytes"
)

// ScanInput will scan Grbl input, stripping out any comments and spaces.
func ScanInput(data []byte, atEOF bool) (advance int, token []byte, err error) {
	adv, tok, err := bufio.ScanLines(data, atEOF)
	if len(tok) == 0 {
		return adv, tok, err
	}

	tok = bytes.ReplaceAll(tok, []byte(" "), nil)

	start := bytes.IndexByte(tok, ';')
	if start > -1 {
		tok = tok[:start]
	}

	for {
		start = bytes.IndexByte(tok, '(')
		if start == -1 {
			return adv, tok, err
		}
		end := bytes.IndexByte(tok[start:], ')')
		if end == -1 {
			return adv, tok[:start], err
		}
		tok = append(tok[:start], tok[start+end+1:]...)
	}
}
