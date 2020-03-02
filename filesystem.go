package tish

import (
	"fmt"
	"os"
)

const (
	dirParent  = ".."
	dirCurrent = "."
)

type Filesystem struct {
	ptr  int
	dirs []string
	root string
	ro   bool
}

func Cwd() (*Filesystem, error) {
  cwd, _ := os.Getwd()
  return RootedFS(cwd)
}

func DefaultFS() (*Filesystem, error) {
	return RootedFS("/")
}

func RootedFS(root string) (*Filesystem, error) {
	i, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !i.IsDir() {
		return nil, fmt.Errorf("%s: not a directory", root)
	}
	fs := Filesystem{
		dirs: make([]string, 1000),
		root: root,
	}
	return &fs, nil
}

func (f *Filesystem) Cwd() string {
	return ""
}

func (f *Filesystem) Chdir(dir string) error {
	return nil
}

func (f *Filesystem) PushDir(dir string) error {
	return nil
}

func (f *Filesystem) PopDir() {

}

func (f *Filesystem) Open(name string) (*os.File, error) {
	return nil, nil
}

func (f *Filesystem) Create(name string) (*os.File, error) {
	if f.ro {
		return nil, fmt.Errorf("filesystem open in read only")
	}
	return nil, nil
}

func (f *Filesystem) OpenFile(name string, flag int, perm int) (*os.File, error) {
	if mode := flag & os.O_RDONLY; mode == 0 {
		return nil, fmt.Errorf("filesystem open in read only")
	}
	return nil, nil
}

func (f *Filesystem) Copy() *Filesystem {
	fs := f

	fs.dirs = make([]string, len(f.dirs))
	copy(fs.dirs, f.dirs)

	return fs
}
