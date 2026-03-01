//go:build !windows

package main

import (
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type unixPTY struct {
	ptmx *os.File
	cmd  *exec.Cmd
}

func newPTY(shell string) (PTY, error) {
	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}
	return &unixPTY{ptmx: ptmx, cmd: cmd}, nil
}

func (p *unixPTY) Read(buf []byte) (int, error)  { return p.ptmx.Read(buf) }
func (p *unixPTY) Write(buf []byte) (int, error) { return p.ptmx.Write(buf) }

func (p *unixPTY) Resize(cols, rows uint16) error {
	return pty.Setsize(p.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

func (p *unixPTY) Close() error {
	p.cmd.Process.Kill()
	p.cmd.Wait()
	return p.ptmx.Close()
}
