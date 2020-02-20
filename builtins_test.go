package tish

import (
	"bytes"
	"fmt"
	"testing"
)

func TestTrue(t *testing.T) {
	b, err := get("true")
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Run(); err != nil {
		t.Fatalf("run true: should return a nil error: %s", err)
	}
}

func TestFalse(t *testing.T) {
	b, err := get("false")
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Run(); err == nil {
		t.Fatal("run false: should return a non-nil error")
	}
}

func TestSeq(t *testing.T) {
	data := []struct {
		Args []string
		Want string
	}{
		{
			Args: []string{"-s", ", ", "--", "5"}, // seq -s ", " 5
			Want: "0, 1, 2, 3, 4, 5\n",
		},
		{
			Args: []string{"-s", ", ", "--", "0"}, // seq -s ", " 0
			Want: "",
		},
		{
			Args: []string{"-s", ", ", "--", "-5"}, // seq -s ", " -5
			Want: "-5, -4, -3, -2, -1, 0\n",
		},
		{
			Args: []string{"-s", ", ", "3", "5"}, // seq -s ", " 3 5
			Want: "3, 4, 5\n",
		},
		{
			Args: []string{"-s", ", ", "--", "-5", "-3"}, // seq -s ", " -5 -3
			Want: "-5, -4, -3\n",
		},
		{
			Args: []string{"-s", ", ", "--", "-3", "-5"}, // seq -s ", " -3 -5
			Want: "-3, -4, -5\n",
		},
		{
			Args: []string{"-s", ", ", "1", "5", "2"}, // seq -s ", " 1 5 2
			Want: "1, 3, 5\n",
		},
		{
			Args: []string{"-s", ", ", "1", "5", "-1"}, // seq -s ", " 1 5 -1
			Want: "",
		},
	}
	for _, d := range data {
		b, err := get("seq")
		if err != nil {
			t.Fatal(err)
		}
		b.args = d.Args
		if err := b.Run(); err != nil {
			t.Fatal("run seq:", err)
		}
		stderr := b.stderr.(*bytes.Buffer)
		if err := stderr.String(); err != "" {
			t.Errorf("expected stderr empty! got %s", err)
			continue
		}
		stdout := b.stdout.(*bytes.Buffer)
		if got := stdout.String(); got != d.Want {
			t.Errorf("values mismatched! want %s; got %s", d.Want, got)
		}
	}
}

func TestPrintf(t *testing.T) {
	t.SkipNow()
}

func TestRandom(t *testing.T) {
	t.SkipNow()
}

func TestHelp(t *testing.T) {
	t.SkipNow()
}

func TestBuiltins(t *testing.T) {
	t.SkipNow()
}

func TestDate(t *testing.T) {
	t.SkipNow()
}

func TestEcho(t *testing.T) {
	t.SkipNow()
}

func get(ident string) (builtin, error) {
	b, ok := builtins[ident]
	if !ok {
		return builtin{}, fmt.Errorf("%s: builtin not found", ident)
	}
	b.stdin = bytes.NewReader(nil)

	var (
		out bytes.Buffer
		err bytes.Buffer
	)
	b.stdout = &out
	b.stderr = &err

	return b, nil
}
