package tish

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

type Shell struct {
	psr *Parser

	env  *Env
	vars *Env

	proc struct {
		exit int
		pid  int
		cmd  string
		args []string
		sys  time.Duration
		user time.Duration
	}
}

func NewShell(r io.Reader) (*Shell, error) {
	p, err := NewParser(r)
	if err != nil {
		return nil, err
	}
	var s Shell
	s.psr = p
	s.env = EmptyEnv()
	s.vars = EmptyEnv()

	return &s, nil
}

func (s *Shell) Execute() (int, error) {
	var (
		cmd Command
		err error
	)
	for err == nil {
		cmd, err = s.psr.Parse()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 1, err
		}
		err = s.execute(cmd)
	}
	return s.proc.exit, err
}

func (s *Shell) execute(cmd Command) error {
	switch cmd := cmd.(type) {
	case List:
		s.executeList(cmd)
	case Simple:
		s.executeSimple(cmd)
	case And:
		s.executeAnd(cmd)
	case Or:
		s.executeOr(cmd)
	case Assign:
		s.executeAssign(cmd)
	case For:
	case Until:
	case While:
	case Case:
	case If:
	default:
		return fmt.Errorf("unsupported command type %T", cmd)
	}
	return nil
}

func (s *Shell) executeList(cmd List) {
	for i := range cmd.cmds {
		s.execute(cmd.cmds[i])
	}
}

func (s *Shell) executeSimple(cmd Simple) {
	env := EnclosedEnv(s.env)
	for _, a := range cmd.env {
		executeAssignWithEnv(a, env)
	}

	name, args := prepare(cmd.words, env)
	if name == "" {
		return
	}
	exe := exec.Command(name, args...)
	exe.Env = env.Environ()
	exe.Stdin = os.Stdin
	exe.Stdout = os.Stdout
	exe.Stderr = os.Stderr

	exe.Run()

	s.proc.cmd = name
	s.proc.args = args
	s.proc.exit = exe.ProcessState.ExitCode()
	s.proc.pid = exe.ProcessState.Pid()
	s.proc.sys = exe.ProcessState.SystemTime()
	s.proc.user = exe.ProcessState.UserTime()
}

func (s *Shell) executeAnd(cmd And) {
	s.execute(cmd.left)
	if s.proc.exit != 0 {
		return
	}
	s.execute(cmd.right)
}

func (s *Shell) executeOr(cmd Or) {
	s.execute(cmd.left)
	if s.proc.exit == 0 {
		return
	}
	s.execute(cmd.right)
}

func (s *Shell) executeAssign(cmd Assign) {
	executeAssignWithEnv(cmd, s.vars)
}

func prepare(words []Word, env *Env) (string, []string) {
	var ws []string
	for _, w := range words {
		ws = append(ws, w.Expand(env))
	}
	if len(ws) == 0 {
		return "", nil
	}
	name := ws[0]
	if len(ws) > 1 {
		return name, ws[1:]
	}
	return name, nil
}

func executeAssignWithEnv(cmd Assign, env *Env) {
	env.Define(cmd.ident.Literal, cmd.word)
}
