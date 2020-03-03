package tish

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type ShellCase struct {
	Input string
	Out   string
	Err   string
	File  string
}

func TestShellScript(t *testing.T) {
	r, errp := os.Open("testdata/script.sh")
	if errp != nil {
		t.Fatalf("fail to open script.sh: %s", errp)
	}
	defer r.Close()

	var (
		in  bytes.Reader
		out bytes.Buffer
		err bytes.Buffer
	)
	fs, errf := Cwd()
	if errf != nil {
		t.Fatalf("init filesystem: %s", errf)
	}
	sh := NewShell(fs, &in, &out, &err)
	sh.uid = 1000
	sh.pid = 12345

	if err := sh.Execute(r); err != nil {
		t.Fatalf("error executing script: %s", err)
	}
	if errc := compareFile(t, "testdata/script.out.sh", out.Bytes()); errc != nil {
		t.Errorf("stdout: %s", errc)
	}
	if errc := compareFile(t, "testdata/script.err.sh", err.Bytes()); errc != nil {
		t.Errorf("stderr: %s", errc)
	}
}

func compareFile(t *testing.T, file string, got []byte) error {
	t.Helper()

	r, err := os.Open(file)
	if err != nil {
		return err
	}
	var (
		want bytes.Buffer
		scan = bufio.NewScanner(r)
	)
	for scan.Scan() {
		line := scan.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if ix := strings.Index(line, "#"); ix > 0 {
			line = strings.TrimSpace(line[:ix])
			// line = line[:ix]
		}
		want.WriteString(line + "\n")
	}
	if err := scan.Err(); err != nil {
		return err
	}
	if !bytes.Equal(want.Bytes(), got) {
		t.Logf("want:\n%s", want.String())
		t.Logf("got:\n%s", string(got))
		err = fmt.Errorf("%s: output mismatched!", file)
	}
	return err
}

func TestShellExecute(t *testing.T) {
	t.Cleanup(func() {
		filepath.Walk("testdata", func(p string, i os.FileInfo, err error) error {
			if i.Mode().IsRegular() && filepath.Ext(p) == ".txt~" {
				os.Remove(p)
			}
			return err
		})
	})
	data := []ShellCase{
		{
			Input: `echo foo bar`,
			Out:   "foo bar",
		},
		{
			Input: `echo -h`,
			Err:   "write arguments to standard output\nusage: echo [-i] [-h] [arg...]",
		},
		{
			Input: `echo -i < testdata/foo.txt`,
			Out:   "foo",
		},
		{
			Input: `alias readfile="echo -i"; readfile < testdata/foo.txt`,
			Out:   "foo",
		},
		{
			Input: `echo`,
			Out:   "",
		},
		{
			Input: `FOO=FOO; BAR=BAR; echo $FOO $BAR`,
			Out:   "FOO BAR",
		},
		{
			Input: `FOO=FOO BAR=BAR env FOO BAR`,
			Out:   "FOO=FOO\nBAR=BAR",
		},
		{
			Input: `echo bar > testdata/bar.txt~`,
			Out:   "bar",
			File:  "testdata/bar.txt~",
		},
		{
			Input: `echo bar >> testdata/bar.txt~`,
			Out:   "bar\nbar",
			File:  "testdata/bar.txt~",
		},
		{
			Input: `echo -h 2> testdata/help.txt~`,
			Out:   "write arguments to standard output\nusage: echo [-i] [-h] [arg...]",
			File:  "testdata/help.txt~",
		},
		{
			Input: `echo foo bar | echo -i`,
			Out:   "foo bar",
		},
		{
			Input: `echo -h | echo -i`,
			Out:   "",
			Err:   "write arguments to standard output\nusage: echo [-i] [-h] [arg...]",
		},
		{
			Input: `echo foo bar | echo -i`,
			Out:   "foo bar",
			Err:   "",
		},
		{
			Input: `echo -h |& echo -i`,
			Out:   "write arguments to standard output\nusage: echo [-i] [-h] [arg...]",
			Err:   "",
		},
		{
			Input: `echo -h 2>&1 | echo -i`,
			Out:   "write arguments to standard output\nusage: echo [-i] [-h] [arg...]",
			Err:   "",
		},
		{
			Input: `echo foobar >&2`,
			Err:   "foobar",
		},
		{
			Input: `echo -h 2>&1`,
			Out:   "write arguments to standard output\nusage: echo [-i] [-h] [arg...]",
		},
		{
			Input: `echo -h > testdata/help.txt~ 2>&1`,
			Out:   "write arguments to standard output\nusage: echo [-i] [-h] [arg...]",
			File:  `testdata/help.txt~`,
		},
		{
			Input: `echo -h  2>&1 > testdata/help.txt~`,
			Out:   "write arguments to standard output\nusage: echo [-i] [-h] [arg...]",
		},
	}
	fs, errf := Cwd()
	if errf != nil {
		t.Fatalf("init filesystem: %s", errf)
	}
	testShellCase(t, fs, data)
}

func testShellCase(t *testing.T, fs *Filesystem, data []ShellCase) {
	t.Helper()

	var (
		sin  = bytes.NewReader(nil)
		sout bytes.Buffer
		serr bytes.Buffer
	)

	trim := func(str string) string {
		return strings.TrimSpace(str)
	}

	for _, d := range data {
		sout.Reset()
		serr.Reset()

		sh := NewShell(fs, sin, &sout, &serr)
		if err := sh.ExecuteString(d.Input); err != nil {
			t.Errorf("%s: unexpected error %s", d.Input, err)
			continue
		}
		if d.File != "" {
			str, err := ioutil.ReadFile(d.File)
			if err != nil {
				t.Errorf("%s: %s", d.Input, err)
				continue
			}
			if got := trim(string(str)); got != d.Out {
				t.Errorf("%s: file mismatched! want %s, got %s", d.Input, d.Out, got)
			}
			continue
		}
		if got := trim(sout.String()); got != d.Out {
			t.Errorf("%s: stdout mismatched! want %s, got %s", d.Input, d.Out, got)
		}
		if got := trim(serr.String()); got != d.Err {
			t.Errorf("%s: stderr mismatched! want %s, got %s", d.Input, d.Err, got)
		}
	}
}
