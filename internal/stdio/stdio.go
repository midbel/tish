package stdio

import (
	"io"
	"os"
)

type pipeWriter struct {
	inner io.Writer
	io.WriteCloser
}

func Pipe(w io.Writer) io.WriteCloser {
	pr, pw, _ := os.Pipe()
	p := &pipeWriter{
		inner:       w,
		WriteCloser: pw,
	}
	go p.copy(pr)
	return p
}

func (p *pipeWriter) copy(r io.ReadCloser) {
	defer r.Close()
	io.Copy(p.inner, r)
}
