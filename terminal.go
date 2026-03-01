package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type PTY interface {
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	Resize(cols, rows uint16) error
	Close() error
}

type Terminal struct {
	ID   string
	Name string
	pty  PTY
}

type TerminalInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var (
	terminals  = make(map[string]*Terminal)
	termMu     sync.RWMutex
	termCount  atomic.Int64
)

func createTerminal() (*TerminalInfo, error) {
	n := termCount.Add(1)
	id := fmt.Sprintf("t%d", n)
	name := fmt.Sprintf("Terminal %d", n)

	p, err := newPTY(config.Shell)
	if err != nil {
		return nil, fmt.Errorf("failed to start terminal: %w", err)
	}

	term := &Terminal{ID: id, Name: name, pty: p}
	termMu.Lock()
	terminals[id] = term
	termMu.Unlock()

	return &TerminalInfo{ID: id, Name: name}, nil
}

func getTerminal(id string) *Terminal {
	termMu.RLock()
	defer termMu.RUnlock()
	return terminals[id]
}

func listTerminals() []TerminalInfo {
	termMu.RLock()
	defer termMu.RUnlock()
	list := make([]TerminalInfo, 0, len(terminals))
	for _, t := range terminals {
		list = append(list, TerminalInfo{ID: t.ID, Name: t.Name})
	}
	return list
}

func closeTerminal(id string) error {
	termMu.Lock()
	term, ok := terminals[id]
	if !ok {
		termMu.Unlock()
		return fmt.Errorf("terminal %s not found", id)
	}
	delete(terminals, id)
	termMu.Unlock()
	return term.pty.Close()
}

func closeAllTerminals() {
	termMu.Lock()
	defer termMu.Unlock()
	for id, t := range terminals {
		t.pty.Close()
		delete(terminals, id)
	}
}

func (t *Terminal) Read(p []byte) (int, error)  { return t.pty.Read(p) }
func (t *Terminal) Write(p []byte) (int, error) { return t.pty.Write(p) }
func (t *Terminal) Resize(cols, rows uint16) error { return t.pty.Resize(cols, rows) }
