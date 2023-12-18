package main

func main() {
	var (
		delim = flag.String("d", "", "delimiter")
		field = flag.Int("f", 0, "field")
	)
	flag.Parse()
}