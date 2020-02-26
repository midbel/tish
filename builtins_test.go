package tish

import (
	"bytes"
	"fmt"
	"testing"
)

func TestBuiltinFuncs(t *testing.T) {
	t.Run("true", testTrue)
	t.Run("false", testFalse)
	t.Run("seq", testSeq)
	t.Run("type", testType)
}

func testType(t *testing.T) {
	data := []struct {
		Args string
		Out  string
		Err  string
	}{
		{Args: "echo", Out: "builtin"},
		{Args: "type", Out: "builtin"},
		{Args: "testdata", Out: "directory"},
		{Args: "README.md", Out: "file"},
		{Args: "foobar", Err: "no such file or directory"},
	}
	for _, d := range data {
		b, err := get("type")
		if err != nil {
			t.Fatal(err)
		}
		b.Args = append(b.Args, d.Args)
		if exit := b.Run(); exit != ExitOk {
			t.Fatalf("run seq: exit code %d", exit)
		}

		var (
			want string
			got  string
		)
		if d.Err != "" {
			stderr := b.Stderr.(*bytes.Buffer)
			got = stderr.String()
			want = fmt.Sprintf("%s: %s\n", d.Args, d.Err)
		} else {
			stdout := b.Stdout.(*bytes.Buffer)
			got = stdout.String()
			want = fmt.Sprintf("%s: %s\n", d.Args, d.Out)
		}
		if got != want {
			t.Errorf("%s: values mismatched! want %s, got %s", d.Args, want, got)
		}
	}
}

func testTrue(t *testing.T) {
	b, err := get("true")
	if err != nil {
		t.Fatal(err)
	}
	if exit := b.Run(); exit != ExitOk {
		t.Fatalf("run true: exit code %d", exit)
	}
}

func testFalse(t *testing.T) {
	b, err := get("false")
	if err != nil {
		t.Fatal(err)
	}
	if exit := b.Run(); exit != ExitVariable {
		t.Fatalf("run false: exit code %d", exit)
	}
}

func testSeq(t *testing.T) {
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
		b.Args = d.Args
		if exit := b.Run(); exit != ExitOk {
			t.Fatalf("run seq: exit code %d", exit)
		}
		stderr := b.Stderr.(*bytes.Buffer)
		if err := stderr.String(); err != "" {
			t.Errorf("expected stderr empty! got %s", err)
			continue
		}
		stdout := b.Stdout.(*bytes.Buffer)
		if got := stdout.String(); got != d.Want {
			t.Errorf("values mismatched! want %s; got %s", d.Want, got)
		}
	}
}

func testPrintf(t *testing.T) {
	t.SkipNow()
}

func testRandom(t *testing.T) {
	t.SkipNow()
}

func testHelp(t *testing.T) {
	t.SkipNow()
}

func testBuiltins(t *testing.T) {
	t.SkipNow()
}

func testDate(t *testing.T) {
	t.SkipNow()
}

func testEcho(t *testing.T) {
	t.SkipNow()
}

func get(ident string) (Builtin, error) {
	b, ok := builtins[ident]
	if !ok {
		return Builtin{}, fmt.Errorf("%s: builtin not found", ident)
	}
	b.Stdin = bytes.NewReader(nil)

	var (
		out bytes.Buffer
		err bytes.Buffer
	)
	b.Stdout = &out
	b.Stderr = &err

	return b, nil
}
