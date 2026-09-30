// Command autodoc-tetris is Tetris as an AutoDoc plugin (AutoDoc's ADR 0209): a dialog over the
// page, drawn through AutoDoc's plugin SDK. AutoDoc runs it from its Plugins menu; run by hand, it
// says so and stops.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yongjohnlee80/autodoc/plugin"

	"github.com/yongjohnlee80/autodoc-tetris/game"
)

func main() {
	if err := plugin.Serve(context.Background(), newTetris()); err != nil {
		fmt.Fprintln(os.Stderr, "autodoc-tetris:", err)
		os.Exit(1)
	}
}

// tetris is the plugin: one game at a time, gravity on its own goroutine, every change a frame.
type tetris struct {
	mu    sync.Mutex
	peer  *plugin.Peer
	g     *game.Game
	w, h  int
	theme plugin.Theme
	best  int
	stop  chan struct{}
	done  chan struct{}
}

func newTetris() *tetris { return &tetris{best: loadBest()} }

func (t *tetris) Open(p *plugin.Peer, o plugin.Open) {
	t.mu.Lock()
	t.peer, t.w, t.h, t.theme = p, o.Width, o.Height, o.Theme
	t.g = game.New(time.Now().UnixNano())
	t.stop, t.done = make(chan struct{}), make(chan struct{})
	t.drawLocked()
	t.mu.Unlock()
	go t.gravity()
}

// gravity ticks the game at its level's pace until Close.
func (t *tetris) gravity() {
	defer close(t.done)
	for {
		t.mu.Lock()
		wait := game.Gravity(t.g.Level)
		t.mu.Unlock()
		select {
		case <-t.stop:
			return
		case <-time.After(wait):
		}
		t.mu.Lock()
		if !t.g.Over && !t.g.Paused {
			t.g.Tick()
			t.drawLocked()
		}
		t.mu.Unlock()
	}
}

func (t *tetris) Key(k plugin.Key) {
	t.mu.Lock()
	defer t.mu.Unlock()
	switch apply(t.g, k) {
	case quit:
		_ = t.peer.Close()
	case restart:
		t.keepBest()
		t.g = game.New(time.Now().UnixNano())
		t.drawLocked()
	case changed:
		t.drawLocked()
	}
}

func (t *tetris) Resize(w, h int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.w, t.h = w, h
	t.drawLocked()
}

func (t *tetris) Theme(th plugin.Theme) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.theme = th
	t.drawLocked()
}

func (t *tetris) Close() {
	close(t.stop)
	<-t.done
	t.mu.Lock()
	defer t.mu.Unlock()
	t.keepBest()
}

func (t *tetris) drawLocked() {
	_ = t.peer.Frame(render(t.g, t.w, t.h, t.theme, t.best))
}

// keepBest saves the game's score when it beats the best.
func (t *tetris) keepBest() {
	if t.g == nil || t.g.Score <= t.best {
		return
	}
	t.best = t.g.Score
	if path := bestPath(); path != "" {
		_ = os.MkdirAll(filepath.Dir(path), 0o700)
		_ = os.WriteFile(path, []byte(strconv.Itoa(t.best)+"\n"), 0o600)
	}
}

// bestPath is where the best score is kept: $XDG_STATE_HOME/autodoc-tetris/best, not the plugin's
// directory, which is a git clone an update resets.
func bestPath() string {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "autodoc-tetris", "best")
}

func loadBest() int {
	b, err := os.ReadFile(bestPath())
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	return n
}
