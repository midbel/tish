package tish

type parser struct {
	scan *Scanner
	curr Token
	peek Token

	err error
}

func Parse(str string) (Word, error) {
	p := parser{
		scan: NewScanner(str),
	}
	p.next()
	p.next()

	return p.Parse()
}

func (p *parser) Parse() (Word, error) {
	return nil, nil
}

func (p *parser) next() {
	p.curr = p.peek
	peek, err := p.scan.Scan()
	if err != nil {
		p.err = err
	}
	p.peek = peek
}

func (p *parser) isDone() bool {
	return p.err != nil && p.curr.Equal(eof)
}
