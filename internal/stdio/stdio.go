package stdio

import (
	"io"
	"os"
)

func Writer(w io.Writer) io.WriteCloser {
	if c, ok := w.(*writer); ok {
		return c
	}
	// pr, pw := io.Pipe()
	pr, pw, _ := os.Pipe()
	drain(pr, w, pr)

	return &writer{
		inner:       w,
		WriteCloser: pw,
	}
}

func Reader(r io.Reader) io.ReadCloser {
	if c, ok := r.(*reader); ok {
		return c
	}
	// pr, pw := io.Pipe()
	pr, pw, _ := os.Pipe()
	drain(r, pw, pw)

	return &reader{
		inner:      r,
		ReadCloser: pr,
	}
}

type writer struct {
	inner io.Writer
	io.WriteCloser
}

type reader struct {
	inner io.Reader
	io.ReadCloser
}

func drain(r io.Reader, w io.Writer, c io.Closer) {
	q := make(chan struct{}, 1)
	go func() {
		defer c.Close()
		q <- struct{}{}
		io.Copy(w, r)
	}()
	<-q
	close(q)
}
