package main

import (
	"os"
	"flag"
)

func main() {
	var (
		list  = flag.Bool("l", false, "print list")
		colorize = flag.Bool("c", false, "print colors")
	)
	flag.Parse()

	dirs, err := os.ReadDir(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

func printList(files []os.DirEntry, colorize bool) {
	for _, f := range files {
		fmt.Println(f.Name())
	}
}

func printName(files []os.DirEntry, colorize bool) {
	for i, f := range files {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Print(f.Name())
	}
	fmt.Println()
}