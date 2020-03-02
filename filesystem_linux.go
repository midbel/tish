package tish

import (
	"os"
)

func checkFile(file string) error {
	i, err := os.Stat(file)
	if err != nil {
		return err
	}
	if mode := i.Mode(); i.IsDir() || mode&0b001001001 == 0 {
		return os.ErrPermission
	}
	return nil
}
