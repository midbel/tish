package tish

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

const MaxHistSize = 500

const separator = "/" //string(path.Separator)

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
	dir, err := f.normalize(root)
	if err != nil {
		return nil, err
	}
	fs, err := RootedFS(dir)
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

func (f *Filesystem) PushDir(step int64) {
	n := len(f.dirs)
	if step > 0 {
		step = int64(n) - (step + 1)
	} else if step < 0 {
		step = -step
	} else {
		if n < 2 {
			return
		}
		n--
		f.dirs[n-1], f.dirs[n] = f.dirs[n], f.dirs[n-1]
		return
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
		n--
		if n >= 0 {
			f.dirs = f.dirs[:n-1]
		}
		return
	}
	if step < 0 || n < int(step) {
		return
	}
	f.dirs = f.dirs[:int(step)]
}

func (f *Filesystem) Expand(file string, nocase bool) ([]string, error) {
	if !hasMeta(file) {
		return []string{file}, nil
	}
	base := f.cwd()
	if path.IsAbs(file) {
		base = f.root
	}
	return f.findFiles(base, strings.Split(file, separator))
}

func (f *Filesystem) findFiles(base string, parts []string) ([]string, error) {
	if len(parts) == 0 {
		return nil, nil
	}
	r, err := f.Open(base)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	is, err := r.Readdir(0)
	if err != nil {
		return nil, err
	}
	var fs []string
	for _, i := range is {
		if !Match(parts[0], i.Name()) {
			continue
		}
		file := path.Join(base, i.Name())
		if i.IsDir() {
			xs, err := f.findFiles(file, parts[1:])
			if err != nil {
				return nil, err
			}
			fs = append(fs, xs...)
		} else {
			if len(parts) == 1 {
				fs = append(fs, file)
			}
		}
	}
	return fs, nil
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
	fs := *f

	fs.dirs = make([]string, len(f.dirs))
	copy(fs.dirs, f.dirs)
	fs.parent = nil

	return &fs
}

func (f *Filesystem) LookPath(name string, paths []string) (string, error) {
	if len(paths) == 0 || strings.Contains(name, separator) {
		if err := checkFile(name); err != nil {
			return name, err
		}
	}
	var err error
	for _, p := range paths {
		n := path.Join(f.root, p, name)
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
			return fmt.Errorf("%s: no such file or directory", path.Base(dir))
		}

		if !i.IsDir() {
			return fmt.Errorf("%s: not a directory", path.Base(dir))
		}
	}

	f.dirs = append(f.dirs, dir)
	return nil
}

func (f *Filesystem) normalize(file string) (string, error) {
	base := f.cwd()
	if path.IsAbs(file) {
		base = f.root
	}
	for _, d := range strings.Split(file, separator) {
		switch d {
		case "..":
			base, _ = path.Split(base)
			if len(base) < len(f.root) {
				return "", fmt.Errorf("%s: no such file or directory", file)
			}
		case ".", "":
		default:
			base = path.Join(base, d)
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

func sameFile(in, out interface{}) error {
	fin, ok := in.(*os.File)
	if !ok {
		return nil
	}
	fout, ok := out.(*os.File)
	if !ok {
		return nil
	}
	var err error
	if fin.Name() == fout.Name() {
		err = fmt.Errorf("%s: already open for reading", fin.Name())
	}
	return err
}

func closeFile(c interface{}) {
	if c, ok := c.(io.Closer); ok {
		c.Close()
	}
}
