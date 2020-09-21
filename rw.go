package tish

import (
	"io"
	"os"
)

type reader struct {
	inner  io.ReadCloser
	writer io.Closer
}

func NewReader(r io.Reader) (io.ReadCloser, error) {
	rs, ws, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	i := reader{
		inner:  rs,
		writer: ws,
	}
	go io.Copy(ws, r)
	return &i, nil
}

func (r *reader) Read(bs []byte) (int, error) {
	return r.inner.Read(bs)
}

func (r *reader) Close() error {
	r.inner.Close()
	return r.writer.Close()
}

type writer struct {
	inner  io.WriteCloser
	reader io.Closer
}

func NewWriter(w io.Writer) (io.WriteCloser, error) {
	rs, ws, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	i := writer{
		inner:  ws,
		reader: rs,
	}
	go io.Copy(w, rs)
	return &i, nil
}

func (w *writer) Write(bs []byte) (int, error) {
	return w.inner.Write(bs)
}

func (w *writer) Close() error {
	w.inner.Close()
	return w.reader.Close()
}
