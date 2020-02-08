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
			Input: "echo; cat; wc; grep",
			Word: makeList(kindSeq,
				makeList(kindSimple, Literal("echo")),
				makeList(kindSimple, Literal("cat")),
				makeList(kindSimple, Literal("wc")),
				makeList(kindSimple, Literal("grep")),
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
		{
			Input: "echo && wc; cat && grep && sort",
			Word: makeList(kindSeq,
				makeList(kindAnd,
					makeList(kindSimple, Literal("echo")),
					makeList(kindSimple, Literal("wc")),
				),
				makeList(kindAnd,
					makeList(kindSimple, Literal("cat")),
					makeList(kindSimple, Literal("grep")),
					makeList(kindSimple, Literal("sort")),
				),
			),
		},
		{
			Input: "echo && wc; cat && grep || sort",
			Word: makeList(kindSeq,
				makeList(kindAnd,
					makeList(kindSimple, Literal("echo")),
					makeList(kindSimple, Literal("wc")),
				),
				makeList(kindOr,
					makeList(kindAnd,
						makeList(kindSimple, Literal("cat")),
						makeList(kindSimple, Literal("grep")),
					),
					makeList(kindSimple, Literal("sort")),
				),
			),
		},
		{
			Input: "echo && wc; cat || grep && sort",
			Word: makeList(kindSeq,
				makeList(kindAnd,
					makeList(kindSimple, Literal("echo")),
					makeList(kindSimple, Literal("wc")),
				),
				makeList(kindAnd,
					makeList(kindOr,
						makeList(kindSimple, Literal("cat")),
						makeList(kindSimple, Literal("grep")),
					),
					makeList(kindSimple, Literal("sort")),
				),
			),
		},
		{
			Input: "echo; cat || grep; sort",
			Word: makeList(kindSeq,
				makeList(kindSimple, Literal("echo")),
				makeList(kindOr,
					makeList(kindSimple, Literal("cat")),
					makeList(kindSimple, Literal("grep")),
				),
				makeList(kindSimple, Literal("sort")),
			),
		},
		{
			Input: `echo $(cat | wc)`,
			Word: makeList(kindSimple,
				Literal("echo"),
				makeList(kindSub,
					makeList(kindPipe, Literal("cat"), Literal("wc")),
				),
			),
		},
		{
			Input: `echo "sum = $(cat ; wc)"`,
			Word: makeList(kindSimple,
				Literal("echo"),
				makeList(kindSimple,
					Literal("sum = "),
					makeList(kindSub,
						makeList(kindSeq, Literal("cat"), Literal("wc")),
					),
				),
			),
		},
	}
	for i, d := range data {
		w, err := Parse(d.Input)
		if err != nil {
			t.Errorf("%d) parsing %s: unexpected error: %s", i+1, d.Input, err)
			continue
		}
		if !d.Word.Equal(w) || d.Word.String() != w.String() {
			t.Errorf("%d) %s: words mismatched!", i+1, d.Input)
			t.Logf("\twant: %s", d.Word)
			t.Logf("\tgot : %s", w)
		}
	}
}

func makeList(kind Kind, ws ...Word) Word {
	if len(ws) == 1 && kind != kindSub {
		return ws[0]
	}
	return List{
		words: ws,
		kind:  kind,
	}
}
