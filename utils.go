package tish

import (
	"io"
	"os"
)

type nopCloser struct {
	io.Writer
}

func NopCloser(w io.Writer) io.WriteCloser {
	return &nopCloser{
		Writer: w,
	}
}

func (_ *nopCloser) Close() error {
	return nil
}

func strArray(str string) []string {
	return []string{str}
}

func openFile(w Word, env Environment) (io.ReadCloser, error) {
	str, err := w.Expand(env)
	if err != nil {
		return nil, err
	}
	if len(str) != 1 {
		return nil, ErrExpansion
	}
	return os.Open(str[0])
}

func writeFile(w Word, env Environment) (io.WriteCloser, error) {
	str, err := w.Expand(env)
	if err != nil {
		return nil, err
	}
	if len(str) != 1 {
		return nil, ErrExpansion
	}
	return os.Create(str[0])
}

func appendFile(w Word, env Environment) (io.WriteCloser, error) {
	str, err := w.Expand(env)
	if err != nil {
		return nil, err
	}
	if len(str) != 1 {
		return nil, ErrExpansion
	}
	return os.OpenFile(str[0], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

type multireader struct {
	rs []io.ReadCloser
	io.Reader
}

func multiReader(rs []io.ReadCloser) io.ReadCloser {
	tmp := make([]io.Reader, len(rs))
	for i := range rs {
		tmp[i] = rs[i]
	}
	return multireader{
		rs:     rs,
		Reader: io.MultiReader(tmp...),
	}
}

func (r multireader) Close() error {
	for _, r := range r.rs {
		r.Close()
	}
	return nil
}

type multiwriter struct {
	ws []io.WriteCloser
	io.Writer
}

func multiWriter(ws []io.WriteCloser) io.WriteCloser {
	tmp := make([]io.Writer, len(ws))
	for i := range ws {
		tmp[i] = ws[i]
	}
	return multiwriter{
		ws:     ws,
		Writer: io.MultiWriter(tmp...),
	}
}

func (w multiwriter) Close() error {
	for _, w := range w.ws {
		w.Close()
	}
	return nil
}
