package tish

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

func Split(str string) []string {
	return SplitFunc(str, func(r rune) bool {
		return r == space || r == tab || r == newline
	})
}

func SplitFunc(str string, fn func(r rune) bool) []string {
	if len(str) == 0 {
		return []string{""}
	}
	return split(strings.NewReader(str), fn)
}

func Words(r io.Reader) []string {
	return split(bufio.NewReader(r), func(r rune) bool {
		return r == space || r == tab || r == newline
	})
}

func split(r io.RuneScanner, fn func(r rune) bool) []string {
	var (
		str []string
		buf bytes.Buffer
	)
	for {
		k, _, err := r.ReadRune()
		if err != nil {
			break
		}
		if fn(k) {
			if buf.Len() > 0 {
				str = append(str, buf.String())
				buf.Reset()
			}
			skipWith(r, fn)
			continue
		}
		buf.WriteRune(k)
	}
	if buf.Len() > 0 {
		str = append(str, buf.String())
	}
	return str
}

func skipWith(r io.RuneScanner, fn func(r rune) bool) {
	for {
		k, _, err := r.ReadRune()
		if err != nil {
			break
		}
		if err != nil || !fn(k) {
			r.UnreadRune()
			break
		}
	}
}
