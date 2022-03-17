package stdio

import (
	"io"
	"os"
)

func Writer(w io.Writer) io.WriteCloser {
	if c, ok := w.(*writer); ok {
		return c
	}
	pr, pw, _ := os.Pipe()
	drainReader(w, pr)

	return &writer{
		inner:       w,
		WriteCloser: pw,
	}
}

func Reader(r io.Reader) io.ReadCloser {
	if c, ok := r.(*reader); ok {
		return c
	}
	pr, pw, _ := os.Pipe()
	drainWriter(r, pw)

	return &reader{
		inner:      r,
		ReadCloser: pr,
	}
}

type writer struct {
	inner io.Writer
	io.WriteCloser
}

func drainReader(w io.Writer, r io.ReadCloser) {
	q := make(chan struct{}, 1)
	go func() {
		defer r.Close()
		q <- struct{}{}
		io.Copy(w, r)
	}()
	<-q
	close(q)
}

type reader struct {
	inner io.Reader
	io.ReadCloser
}

func drainWriter(r io.Reader, w io.WriteCloser) {
	q := make(chan struct{}, 1)
	go func() {
		defer w.Close()
		q <- struct{}{}
		io.Copy(w, r)
	}()
	<-q
	close(q)
}
