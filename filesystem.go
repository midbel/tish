package tish

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const MaxHistSize = 500

const separator = string(filepath.Separator)

type Filesystem struct {
	dirs []string
	root string
	ro   bool

	parent *Filesystem
}

func Cwd() (*Filesystem, error) {
	cwd, _ := os.Getwd()
	return RootedFS(cwd)
}

func DefaultFS() (*Filesystem, error) {
	return RootedFS(separator)
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
		dirs: make([]string, 0, MaxHistSize),
		root: root,
	}
	return &fs, fs.chdir(fs.root)
}

func (f *Filesystem) Reset() error {
	f.dirs = f.dirs[:0]
	return f.chdir(f.root)
}

func (f *Filesystem) Chdir(dir string) error {
	switch dir {
	case "-":
		if n := len(f.dirs); n > 0 {
			f.chdir(f.dirs[n-1])
		}
		return nil
	case separator:
		return f.chdir(f.root)
	default:
	}

	file, err := f.normalize(dir)
	if err == nil {
		err = f.chdir(file)
	}
	return err
}

func (f *Filesystem) Chroot(root string) (*Filesystem, error) {
	if err := f.Chdir(root); err != nil {
		return nil, err
	}
	fs, err := RootedFS(f.cwd())
	if err != nil {
		return nil, err
	}
	fs.parent = f
	return fs, nil
}

func (f *Filesystem) Cwd() string {
	str := strings.TrimPrefix(f.cwd(), f.root)
	if !strings.HasPrefix(str, separator) {
		str = separator + str
	}
	return str
}

func (f *Filesystem) Dirs() []string {
	dirs := make([]string, len(f.dirs))
	for i, j := 0, len(dirs)-1; i < len(dirs); i, j = i+1, j-1 {
		dirs[i] = strings.TrimPrefix(f.dirs[j], f.root)
		if !strings.HasPrefix(dirs[i], separator) {
			dirs[i] = separator + dirs[i]
		}
	}
	return dirs
}

func (f *Filesystem) SwitchHead() {
	n := len(f.dirs)
	if n < 2 {
		return
	}
	n--
	f.dirs[n-1], f.dirs[n] = f.dirs[n], f.dirs[n-1]
}

func (f *Filesystem) PopHead() {
	n := len(f.dirs)
	if n == 0 {
		return
	}
	f.dirs = f.dirs[:n-1]
}

func (f *Filesystem) PushDir(step int64) {
	n := len(f.dirs)
	if step > 0 {
		step = int64(n) - (step + 1)
	} else if step < 0 {
		step = -step
	} else {
		return // do nothing for now
	}
	if step < 0 || n < int(step) {
		return
	}
	first, last := f.dirs[:int(step)], f.dirs[int(step):]
	f.dirs = append(last, first...)
}

func (f *Filesystem) PopDir(step int64) {
	n := len(f.dirs)
	if step > 0 {
		step = int64(n) - step
	} else if step < 0 {
		step = -step
	} else {
		return // do nothing for now
	}
	if step < 0 || n < int(step) {
		return
	}
	f.dirs = f.dirs[:int(step)]
}

func (f *Filesystem) Open(name string) (*os.File, error) {
	file, err := f.normalize(name)
	if err != nil {
		return nil, err
	}
	return os.Open(file)
}

func (f *Filesystem) Create(name string) (*os.File, error) {
	if f.ro {
		return nil, fmt.Errorf("filesystem open in read only")
	}
	file, err := f.normalize(name)
	if err != nil {
		return nil, err
	}
	return os.Create(file)
}

func (f *Filesystem) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	if mode := flag & os.O_RDONLY; f.ro && mode == 0 {
		return nil, fmt.Errorf("filesystem open in read only")
	}
	file, err := f.normalize(name)
	if err != nil {
		return nil, err
	}
	return os.OpenFile(file, flag, perm)
}

func (f *Filesystem) Copy() *Filesystem {
	fs := f

	fs.dirs = make([]string, len(f.dirs))
	copy(fs.dirs, f.dirs)

	return fs
}

func (f *Filesystem) LookPath(name string, paths []string) (string, error) {
	if len(paths) == 0 || strings.Contains(name, separator) {
		if err := checkFile(name); err != nil {
			return name, err
		}
	}
	var err error
	for _, p := range paths {
		n := filepath.Join(f.root, p, name)
		if err = checkFile(n); err == nil {
			name = n
			break
		}
	}
	return name, err
}

func (f *Filesystem) chdir(dir string) error {
	if dir != separator {
		i, err := os.Stat(dir)
		if err != nil {
			return fmt.Errorf("%s: no such file or directory", filepath.Base(dir))
		}

		if !i.IsDir() {
			return fmt.Errorf("%s: not a directory", filepath.Base(dir))
		}
	}

	f.dirs = append(f.dirs, dir)
	return nil
}

func (f *Filesystem) normalize(file string) (string, error) {
	base := f.cwd()
	if filepath.IsAbs(file) {
		base = f.root
	}
	for _, d := range strings.Split(file, separator) {
		switch d {
		case "..":
			base, _ = filepath.Split(base)
			if len(base) < len(f.root) {
				return "", fmt.Errorf("%s: no such file or directory", file)
			}
		case ".", "":
		default:
			base = filepath.Join(base, d)
		}
	}
	return base, nil
}

func (f *Filesystem) cwd() string {
	n := len(f.dirs)
	if n == 0 {
		return f.root
	}
	return f.dirs[n-1]
}
