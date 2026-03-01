//go:build windows

package main

import (
	"io"
	"os"
	"os/exec"
)

type windowsPTY struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader io.ReadCloser
	writer *io.PipeWriter
}

func newPTY(shell string) (PTY, error) {
	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		cmd.Wait()
		pw.Close()
	}()

	return &windowsPTY{cmd: cmd, stdin: stdin, reader: pr, writer: pw}, nil
}

func (p *windowsPTY) Read(buf []byte) (int, error)  { return p.reader.Read(buf) }
func (p *windowsPTY) Write(buf []byte) (int, error) { return p.stdin.Write(buf) }
func (p *windowsPTY) Resize(cols, rows uint16) error { return nil }

func (p *windowsPTY) Close() error {
	p.stdin.Close()
	p.cmd.Process.Kill()
	p.writer.Close()
	p.reader.Close()
	return nil
}
