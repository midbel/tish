package tish

type parser struct {
  scan *Scanner
  curr Token
  peek Token
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
  p.peek = p.scan.Scan()
}

func (p *parser) isDone() bool {
  return p.curr.Equal(eof)
}
