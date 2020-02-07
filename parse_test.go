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
			Input: "find | cat | grep",
			Word:  makeList(kindPipe, Literal("find"), Literal("cat"), Literal("grep")),
		},
		{
			Input: "find | cat | grep; wc",
			Word: makeList(kindSeq,
				makeList(kindPipe, Literal("find"), Literal("cat"), Literal("grep")),
				makeList(kindSimple, Literal("wc")),
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
			Word: makeList(kindOr,
				makeList(kindSimple, Literal("echo"), Variable("BAR")),
				makeList(kindSimple, Literal("echo"), Variable("FOO")),
			),
		},
		{
			Input: "echo; echo | cat",
			Word: makeList(kindSeq,
				makeList(kindSimple, Literal("echo")),
				makeList(kindPipe,
					makeList(kindSimple, Literal("echo")),
					makeList(kindSimple, Literal("cat")),
				),
			),
		},
		{
			Input: "echo; echo $FOO && echo",
			Word: makeList(kindSeq,
				makeList(kindSimple, Literal("echo")),
				makeList(kindAnd,
					makeList(kindSimple, Literal("echo"), Variable("FOO")),
					makeList(kindSimple, Literal("echo")),
				),
			),
		},
		{
			Input: "echo | cat; echo",
			Word: makeList(kindSeq,
				makeList(kindPipe,
					makeList(kindSimple, Literal("echo")),
					makeList(kindSimple, Literal("cat")),
				),
				makeList(kindSimple, Literal("echo")),
			),
		},
		{
			Input: "echo | cat && echo", // as (echo foo | echo) && echo bar
			Word: makeList(kindAnd,
				makeList(kindPipe,
					makeList(kindSimple, Literal("echo")),
					makeList(kindSimple, Literal("cat")),
				),
				makeList(kindSimple, Literal("echo")),
			),
		},
		{
			Input: "echo || echo | cat", // as echo foo || (echo bar | cat)
			Word: makeList(kindOr,
				makeList(kindSimple, Literal("echo")),
				makeList(kindPipe,
					makeList(kindSimple, Literal("echo")),
					makeList(kindSimple, Literal("cat")),
				),
			),
		},
	}
	for i, d := range data {
		w, err := Parse(d.Input)
		if err != nil {
			t.Errorf("%d) parsing %s: unexpected error (%s)", i+1, d.Input, err)
			continue
		}
		if !d.Word.Equal(w) || d.Word.String() != w.String() {
			t.Errorf("%d) %s: words mismatched! want %s, got %s", i+1, d.Input, d.Word, w)
		}
	}
}

func makeList(kind Kind, ws ...Word) Word {
	if len(ws) == 1 {
		return ws[0]
	}
	return List{
		words: ws,
		kind:  kind,
	}
}
