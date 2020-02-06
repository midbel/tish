package tish

import (
	"testing"
)

func TestParse(t *testing.T) {
	data := []struct {
		Input string
		Word  Word
	}{
		{
			Input: "echo",
			Word:  makeList(kindSimple, Literal("echo")),
		},
		{
			Input: "echo foobar",
			Word:  makeList(kindSimple, Literal("echo"), Literal("foobar")),
		},
		{
			Input: "echo \"foobar\"",
			Word:  makeList(kindSimple, Literal("echo"), Literal("foobar")),
		},
		{
			Input: "echo $FOO",
			Word:  makeList(kindSimple, Literal("echo"), Variable("FOO")),
		},
		{
			Input: "echo foo; echo bar",
			Word: makeList(kindSeq,
				makeList(kindSimple, Literal("echo"), Literal("foo")),
				makeList(kindSimple, Literal("echo"), Literal("bar")),
			),
		},
		{
			Input: "echo foo; echo $FOO",
			Word: makeList(kindSeq,
				makeList(kindSimple, Literal("echo"), Literal("foo")),
				makeList(kindSimple, Literal("echo"), Variable("FOO")),
			),
		},
		{
			Input: "echo foo | echo",
			Word: makeList(kindPipe,
				makeList(kindSimple, Literal("echo"), Literal("foo")),
				makeList(kindSimple, Literal("echo")),
			),
		},
		{
			Input: "echo foo && echo $FOO",
			Word: makeList(kindAnd,
				makeList(kindSimple, Literal("echo"), Literal("foo")),
				makeList(kindSimple, Literal("echo"), Variable("FOO")),
			),
		},
		{
			Input: "echo $BAR || echo $FOO",
			Word: makeList(kindPipe,
				makeList(kindSimple, Literal("echo"), Literal("BAR")),
				makeList(kindSimple, Literal("echo"), Variable("FOO")),
			),
		},
		{
			Input: "echo foo; echo $FOO | echo",
			Word: makeList(kindSeq,
				makeList(kindSimple, Literal("echo"), Literal("foo")),
				makeList(kindPipe,
					makeList(kindSimple, Literal("echo"), Variable("FOO")),
					makeList(kindSimple, Literal("echo")),
				),
			),
		},
	}
	for _, d := range data {
		w, err := Parse(d.Input)
		if err != nil {
			t.Errorf("parse: unexpected error: %s", err)
			continue
		}
		if !d.Word.Equal(w) || d.Word.String() != w.String() {
			t.Errorf("words are not equal! want %s, got %s", w, d.Word)
		}
	}
}

func makeList(kind Kind, ws ...Word) Word {
	return List{
		words: ws,
		kind:  kind,
	}
}
