package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	unixDate = "%a %b %k %T %Z %Y" // Mon Jan _2 15:04:05 MST 2006
	isoDate  = "%Y-%m-%dT%T%z"
)

var mapping = map[string]string{
	"%":   "%",
	"H":   "15",
	"I":   "03",
	"k":   "3",
	"M":   "04",
	"p":   "PM",
	"N":   "000000000",
	"R":   "15:04",
	"s":   "",
	"S":   "05",
	"T":   "15:04:05",
	"z":   "-0700",
	":z":  "-07:00",
	"::z": "-07:00:00",
	"Z":   "MST",
	"a":   "Mon",
	"A":   "Monday",
	"b":   "Jan",
	"B":   "January",
	"c":   "Mon Jan 3 15:04:05 2006",
	"C":   "",
	"d":   "02",
	"D":   "01/02/06",
	"e":   "2",
	"F":   "",
	"g":   "",
	"G":   "",
	"h":   "Jan",
	"j":   "002",
	"m":   "01",
	"u":   "",
	"U":   "",
	"V":   "",
	"w":   "",
	"W":   "",
	"x":   "",
	"y":   "06",
	"Y":   "2006",
}

func main() {
	var (
		format = flag.String("d", unixDate, "format")
		iso    = flag.Bool("i", false, "iso format")
		utc    = flag.Bool("u", false, "utc")
	)
	flag.Parse()

	if *iso {
		*format = isoDate
	}

	pat, err := parseFormat(*format)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	now := time.Now()
	if *utc {
		now = now.UTC()
	}
	fmt.Fprintln(os.Stderr, now.Format(pat))
}

func parseFormat(str string) (string, error) {
	var (
		rs = strings.NewReader(str)
		ws strings.Builder
	)
	for rs.Len() > 0 {
		r, _, err := rs.ReadRune()
		if err != nil {
			return "", err
		}
		if r != '%' {
			ws.WriteRune(r)
			continue
		}
		if r, _, err = rs.ReadRune(); err != nil {
			return "", err
		}
		pat, ok := mapping[string(r)]
		if !ok {
			pat = string(r)
		}
		ws.WriteString(pat)
	}
	return ws.String(), nil
}
