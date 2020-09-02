package tish

import (
  "io"
)

type Parser struct {
  scan *Scanner
  curr Token
  peek Token
}

func Parse(r io.Reader) (*Parser, error) {
  s, err := NewScanner(r)
  if err != nil {
    return nil, err
  }

  var p Parser
  p.scan = s
  p.next()
  p.next()

  return &p, nil
}

func (p *Parser) Parse() error {
  return nil
}

func (p *Parser) next() {
  p.curr = p.peek
  p.peek = p.scan.Next()
}

func (p *Parser) isDone() {
  return p.curr.Type == TokEOF
}
