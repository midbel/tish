package words_test

import (
	"testing"

	"github.com/midbel/tish"
	"github.com/midbel/tish/internal/words"
)

func TestExpander(t *testing.T) {
	data := []struct {
		Name string
		words.Expander
		Want []string
	}{
		{
			Name:     "slice",
			Expander: createSlice("foobar", 0, 3),
			Want:     []string{"foo"},
		},
		{
			Name:     "slice",
			Expander: createSlice("foobar", 0, 10),
			Want:     []string{"foobar"},
		},
		{
			Name:     "slice",
			Expander: createSlice("foobar", 3, 0),
			Want:     []string{"bar"},
		},
		{
			Name:     "slice",
			Expander: createSlice("foobar", 3, 3),
			Want:     []string{"bar"},
		},
		{
			Name:     "slice",
			Expander: createSlice("foobar", 3, -3),
			Want:     []string{"foo"},
		},
		{
			Name:     "slice",
			Expander: createSlice("foobar", 3, 10),
			Want:     []string{"bar"},
		},
		{
			Name:     "slice",
			Expander: createSlice("foobar", 10, 10),
			Want:     []string{""},
		},
		{
			Name:     "slice",
			Expander: createSlice("foobar", -3, 0),
			Want:     []string{"bar"},
		},
		{
			Name:     "slice",
			Expander: createSlice("foobar", 0, -3),
			Want:     []string{"bar"},
		},
		{
			Name:     "list-brace",
			Expander: createListBrace("pre-", "-post", "foo", "bar"),
			Want:     []string{"pre-foo-post", "pre-bar-post"},
		},
		{
			Name:     "list-brace",
			Expander: createListBrace("", "-post", "foo", "bar"),
			Want:     []string{"foo-post", "bar-post"},
		},
		{
			Name:     "list-brace",
			Expander: createListBrace("pre-", "", "foo", "bar"),
			Want:     []string{"pre-foo", "pre-bar"},
		},
		{
			Name:     "range-brace",
			Expander: createRangeBrace(1, 3, 1, "pre-", "-post"),
			Want:     []string{"pre-1-post", "pre-2-post", "pre-3-post"},
		},
	}
	env := tish.EmptyEnv()
	env.Define("foobar", []string{"foobar"})
	for i, d := range data {
		t.Run(d.Name, func(t *testing.T) {
			got, err := d.Expand(env, false)
			if err != nil {
				t.Fatalf("unexpected error expanding foobar! %s", err)
			}
			if len(got) != len(d.Want) {
				t.Fatalf("length mismatched! want %d, got %d", len(d.Want), len(got))
			}
			for j := range d.Want {
				if d.Want[j] != got[j] {
					t.Errorf("%d) strings mismatched! want %s, got %s", i+1, d.Want[j], got[j])
				}
			}
		})
	}
}

func createRangeBrace(from, to, step int, prefix, suffix string) words.Expander {
	if step == 0 {
		step = 1
	}
	b := words.ExpandRangeBrace{
		From: from,
		To:   to,
		Step: step,
	}
	if prefix != "" {
		b.Prefix = createWord(prefix)
	}
	if suffix != "" {
		b.Suffix = createWord(suffix)
	}
	return b
}

func createListBrace(prefix, suffix string, list ...string) words.Expander {
	var b words.ExpandListBrace
	for i := range list {
		b.Words = append(b.Words, createWord(list[i]))
	}
	if prefix != "" {
		b.Prefix = createWord(prefix)
	}
	if suffix != "" {
		b.Suffix = createWord(suffix)
	}
	return b
}

func createWord(word string) words.Expander {
	w := words.ExpandWord{
		Literal: word,
	}
	return w
}

func createSlice(ident string, off, siz int) words.Expander {
	return words.ExpandSlice{
		Ident:  ident,
		Offset: off,
		Size:   siz,
	}
}
