package main

import (
  "bufio"
  "flag"
  "fmt"
  "os"
  "strings"
)

func main() {
  flag.Parse()
  if flag.NArg() > 0 {
    fmt.Println(strings.Join(flag.Args(), " "))
    return
  }
  s := bufio.NewScanner(os.Stdin)
  for s.Scan() {
    fmt.Println(s.Text())
  }
  if err := s.Err(); err != nil {
    os.Exit(1)
  }
}
