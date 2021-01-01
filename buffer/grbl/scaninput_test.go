package grbl

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScanInput(t *testing.T) {
	const input = `
code
; command
line ; with middle comment

and one (with center) comment

and a (broken ; one)
yet another ( broken one
`
	s := bufio.NewScanner(strings.NewReader(input))
	s.Split(ScanInput)

	var lines []string
	for s.Scan() {
		if s.Text() == "" {
			continue
		}
		lines = append(lines, s.Text())
	}

	assert.Equal(t, []string{"code", "line", "andonecomment", "anda", "yetanother"}, lines)

}
