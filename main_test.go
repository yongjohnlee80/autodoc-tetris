package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yongjohnlee80/autodoc/plugin"

	"github.com/yongjohnlee80/autodoc-tetris/game"
)

func text(f *plugin.Frame) string {
	var b strings.Builder
	for _, row := range f.Rows() {
		for _, r := range row {
			b.WriteString(r.Text)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// styleAt is the style of the frame's cell (x, y).
func styleAt(f *plugin.Frame, x, y int) plugin.Style {
	col := 0
	for _, r := range f.Rows()[y] {
		n := len([]rune(r.Text))
		if x < col+n {
			return r.Style
		}
		col += n
	}
	return plugin.Style{}
}

// scoreOf reads the panel's score: the last field of the row under SCORE.
func scoreOf(frame string) int {
	lines := strings.Split(frame, "\n")
	for i, l := range lines {
		if strings.Contains(l, "SCORE") && i+1 < len(lines) {
			f := strings.Fields(lines[i+1])
			if len(f) > 0 {
				n, _ := strconv.Atoi(f[len(f)-1])
				return n
			}
		}
	}
	return -1
}

var dark = plugin.Theme{Name: "dark", Colors: map[string]string{"document.lineNumber": "#5f5f5f"}}

// TestTheFrameIsTheBoardThePieceItsGhostAndThePanel: the walls and floor, the falling piece in its
// colour, its ghost at the bottom, the next piece and the counts.
func TestTheFrameIsTheBoardThePieceItsGhostAndThePanel(t *testing.T) {
	g := game.New(7)
	g.Cur = game.Piece{Kind: game.T, X: 3, Y: game.Hidden}
	g.Score, g.Level, g.Lines = 1234, 3, 21
	f := render(g, needW, needH, dark, 5000)
	s := text(f)
	for _, want := range []string{"│", "└──", "NEXT", "SCORE", "1234", "LEVEL", "LINES", "21", "BEST", "5000", "Spc  hard drop"} {
		if !strings.Contains(s, want) {
			t.Errorf("the frame lacks %q:\n%s", want, s)
		}
	}
	// T's nub at board column 4, row 0: past the margin and the wall, screen column 2+4*2
	if st := styleAt(f, 10, 0); st.FG != "magenta" {
		t.Errorf("T's cell is %+v, want magenta:\n%s", st, s)
	}
	ghost := g.Ghost()
	gx, gy := ghost.Cells()[0][0], ghost.Cells()[0][1]-game.Hidden
	if st := styleAt(f, 2+gx*2, gy); st.FG != "#5f5f5f" || !strings.Contains(strings.Split(s, "\n")[gy], "░░") {
		t.Errorf("the ghost at (%d, %d) is %+v:\n%s", gx, gy, st, s)
	}
}

// TestMonoDrawsEveryPieceBrightWhite: a theme without colour gets its brightest white.
func TestMonoDrawsEveryPieceBrightWhite(t *testing.T) {
	p := paletteFor(plugin.Theme{Name: "mono"})
	for _, k := range game.Kinds {
		if p.piece[k].FG != "brightwhite" {
			t.Errorf("%v in mono: %q", k, p.piece[k].FG)
		}
	}
	if paletteFor(dark).piece[game.L].FG != "#ff8700" {
		t.Error("L is not orange where there is colour")
	}
}

// TestASmallWindowSaysWhatItNeeds, and pause and game over say so over the board.
func TestASmallWindowSaysWhatItNeeds(t *testing.T) {
	g := game.New(1)
	if s := text(render(g, 30, 10, dark, 0)); !strings.Contains(s, "Tetris needs a window") {
		t.Errorf("a small window:\n%s", s)
	}
	g.TogglePause()
	if s := text(render(g, needW, needH, dark, 0)); !strings.Contains(s, "PAUSED") {
		t.Errorf("paused:\n%s", s)
	}
	g.TogglePause()
	g.Over = true
	if s := text(render(g, needW, needH, dark, 0)); !strings.Contains(s, "GAME OVER") || !strings.Contains(s, "Enter: again") {
		t.Errorf("over:\n%s", s)
	}
}

// TestTheKeysPlayTheGame: the arrows and Vim's keys move and turn, Space drops, p pauses, q quits,
// Enter restarts only a lost game, and a key that changes nothing says so.
func TestTheKeysPlayTheGame(t *testing.T) {
	g := game.New(3)
	g.Cur = game.Piece{Kind: game.T, X: 3, Y: 5}
	steps := []struct {
		key  string
		want action
		ok   func() bool
	}{
		{"Left", changed, func() bool { return g.Cur.X == 2 }},
		{"l", changed, func() bool { return g.Cur.X == 3 }},
		{"x", changed, func() bool { return g.Cur.Rot == 1 }},
		{"z", changed, func() bool { return g.Cur.Rot == 0 }},
		{"Down", changed, func() bool { return g.Cur.Y == 6 }},
		{"F5", nothing, func() bool { return true }},
		{"Enter", nothing, func() bool { return true }},
		{"p", changed, func() bool { return g.Paused }},
		{"Left", nothing, func() bool { return g.Cur.X == 3 }},
		{"p", changed, func() bool { return !g.Paused }},
		{" ", changed, func() bool { return g.Score > 0 && g.Cur.Y < 5 }},
		{"q", quit, func() bool { return true }},
	}
	for i, s := range steps {
		if got := apply(g, plugin.Key{Key: s.key}); got != s.want || !s.ok() {
			t.Fatalf("step %d, %q: %v (want %v), state %+v", i, s.key, got, s.want, g.Cur)
		}
	}
	g.Over = true
	if apply(g, plugin.Key{Key: "Enter"}) != restart {
		t.Error("Enter on a lost game does not restart it")
	}
}

// TestThePluginSpeaksTheProtocol: the built binary, run as AutoDoc runs it, answers the handshake,
// draws the board, takes a key, keeps its best score where the state goes, and exits on close.
func TestThePluginSpeaksTheProtocol(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "autodoc-tetris")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	toPlugin, hostOut, _ := os.Pipe()
	hostIn, fromPlugin, _ := os.Pipe()
	state := t.TempDir()
	cmd := exec.Command(bin)
	cmd.Stdin, cmd.Stdout = toPlugin, fromPlugin
	cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+state)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	toPlugin.Close()
	fromPlugin.Close()
	got := make(chan string, 256)
	ctx := context.Background()
	link, err := plugin.NewLink(ctx, plugin.FileConn(hostIn, hostOut), func(m string, p []any) {
		if m == plugin.MethodFrame {
			rows, _, _ := plugin.ReadFrame(p)
			var b strings.Builder
			for _, r := range rows {
				for _, run := range r {
					b.WriteString(run.Text)
				}
				b.WriteByte('\n')
			}
			got <- b.String()
			return
		}
		got <- m
	})
	if err != nil {
		t.Fatal(err)
	}
	defer link.Close()
	wait := func(what func(string) bool) string {
		t.Helper()
		deadline := time.After(10 * time.Second)
		for {
			select {
			case s := <-got:
				if what(s) {
					return s
				}
			case <-deadline:
				t.Fatalf("never came; stderr: %s", stderr.String())
			}
		}
	}
	_ = link.Notify(ctx, plugin.MethodOpen, plugin.OpenParams(plugin.Open{Protocol: plugin.Protocol, Width: needW, Height: needH, Theme: dark}))
	wait(func(s string) bool { return s == plugin.MethodReady })
	wait(func(s string) bool { return strings.Contains(s, "NEXT") && strings.Contains(s, "SCORE") })
	_ = link.Notify(ctx, plugin.MethodKey, plugin.KeyParams(plugin.Key{Key: " ", Text: " "}))
	wait(func(s string) bool { return scoreOf(s) > 0 }) // a hard drop scores 2 a row
	_ = link.Notify(ctx, plugin.MethodClose, plugin.EmptyParams())
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("the plugin exited with %v: %s", err, stderr.String())
		}
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("the plugin did not exit on plugin.close")
	}
	if b, err := os.ReadFile(filepath.Join(state, "autodoc-tetris", "best")); err != nil || strings.TrimSpace(string(b)) == "0" {
		t.Errorf("the best score was not kept: %q, %v", b, err)
	}
}
