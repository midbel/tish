package tish

import (
	"testing"
)

func TestParse(t *testing.T) {
	data := []struct {
		Input string
		Words []Word
	}{
		{
			Input: "echo",
			Words: []Word{
				Literal("echo"),
			},
		},
		{
			Input: "echo foobar",
			Words: []Word{
				Literal("echo"),
				Literal("foobar"),
			},
		},
		{
			Input: "echo \"foobar\"",
			Words: []Word{
				Literal("echo"),
				Literal("foobar"),
			},
		},
		{
			Input: "echo $FOO",
			Words: []Word{
				Literal("echo"),
				Variable("FOO"),
			},
		},
	}
	for _, d := range data {
		w, err := Parse(d.Input)
		if err != nil {
			t.Errorf("parse: unexpected error: %s", err)
			continue
		}
		is := List{words: d.Words}
		if !is.Equal(w) {
			t.Errorf("Word are not equal! want %s, got %s", w, is)
		}
	}
}
