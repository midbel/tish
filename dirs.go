package tish

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/midbel/tish/internal/stack"
)

type Stack interface {
	Cwd() string
	Dirs() []string

	Chdir(string) error
	Pushd(string) error
	Popd(string) error
}

const (
	dirCurr   = "."
	dirParent = ".."
	dirOld    = "-"
)

type dirsstack struct {
	list stack.Stack[string]
}

func DirectoryStack() Stack {
	return &dirsstack{
		list: stack.New[string](),
	}
}

func (d *dirsstack) Cwd() string {
	return d.list.Curr()
}

func (d *dirsstack) Dirs() []string {
	var list []string
	for i := d.list.Len() - 1; i >= 0; i-- {
		list = append(list, d.list.At(i))
	}
	return list
}

func (d *dirsstack) Chdir(dir string) error {
	switch dir {
	case dirCurr:
	case dirParent:
		dir = d.list.Curr()
		d.list.Push(filepath.Dir(dir))
	case dirOld:
		d.list.Pop()
	case "":
		// back to home dir
	default:
		i, err := os.Stat(dir)
		if err != nil {
			return err
		}
		if !i.IsDir() {
			return fmt.Errorf("%s: not a directory", dir)
		}
		d.list.Push(dir)
	}
	return nil
}

func (d *dirsstack) Pushd(dir string) error {
	var (
		off int
		err error
	)
	switch {
	case strings.HasPrefix(dir, "+"):
		off, err = strconv.Atoi(dir)
		if err == nil {
			d.list.RotateLeft(off)
		}
	case strings.HasPrefix(dir, "-"):
		off, err = strconv.Atoi(dir)
		if err == nil {
			d.list.RotateRight(-off)
		}
	default:
		err = d.Chdir(dir)
	}
	return err
}

func (d *dirsstack) Popd(dir string) error {
	var (
		off int
		err error
	)
	switch {
	case strings.HasPrefix(dir, "+"):
		off, err = strconv.Atoi(dir)
		if err == nil {
			d.list.RemoveLeft(off)
		}
	case strings.HasPrefix(dir, "-"):
		off, err = strconv.Atoi(dir)
		if err == nil {
			d.list.RemoveRight(off)
		}
	default:
		d.list.Pop()
	}
	return err
}
