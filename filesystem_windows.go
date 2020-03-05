package tish

import (
	"os"
	"path"
)

var cmdexts = []string{".exe", ".bat", ".cmd"}

func checkFile(file string) error {
	i, err := os.Stat(file)
	if err != nil {
		return err
	}
	if !i.Mode().IsRegular() {
		return os.ErrPermission
	}
	for i, e := 0, path.Ext(file); i < len(cmdexts); i++ {
		if e == cmdexts[i] {
			return nil
		}
	}
	return os.ErrPermission
}
