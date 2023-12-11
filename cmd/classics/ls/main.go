package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var (
		list     = flag.Bool("l", false, "print list")
		colorize = flag.Bool("c", false, "print colors")
		print    = printName
	)
	flag.Parse()

	dirs, err := os.ReadDir(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *list {
		print = printList
	}
	print(dirs, *colorize)
}

func printList(files []os.DirEntry, colorize bool) error {
	for _, f := range files {
		i, err := f.Info()
		if err != nil {
			return err
		}
		w := i.ModTime().Format("2006-01-02 15:04")
		fmt.Printf("%s  %12d  %s  %s", f.Type(), i.Size(), w, f.Name())
		fmt.Println()
	}
	return nil
}

func printName(files []os.DirEntry, colorize bool) error {
	for i, f := range files {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Print(f.Name())
	}
	fmt.Println()
	return nil
}
