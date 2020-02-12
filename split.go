package tish

import (
	"io"
	"strings"
)

func Split(str string) ([]string, error) {
	var words []string

	s := NewScanner(strings.NewReader(str))
	for {
		tok, err := s.Scan()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if tok.Equal(eof) {
			break
		}
		if tok.Equal(blank) {
			continue
		}
		words = append(words, tok.Literal)
	}
	return words, nil
}
