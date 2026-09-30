package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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

// scoreOf reads the panel's score: the number after SCORE, on its row.
func scoreOf(frame string) int {
	for _, l := range strings.Split(frame, "\n") {
		if _, after, ok := strings.Cut(l, "SCORE"); ok {
			if f := strings.Fields(after); len(f) > 0 {
				n, _ := strconv.Atoi(f[0])
				return n
			}
		}
	}
	return -1
}

var dark = plugin.Theme{Name: "dark", Colors: map[string]string{"document.lineNumber": "#5f5f5f", "document.cursor": "#ffaf00"}}

// TestTheGameIsCalebAndEliasTetris: the title, 갈렙과 엘리아 테트리스, at the top of the panel, in
// the theme's accent, bold, above the next piece and the scores.
func TestTheGameIsCalebAndEliasTetris(t *testing.T) {
	g := game.New(1, game.MinHeight)
	f := render(g, needW, needH, dark, 0, 0)
	rows := strings.Split(text(f), "\n")
	if !strings.Contains(rows[0], "갈렙과 엘리아") || !strings.Contains(rows[1], "테트리스") {
		t.Fatalf("the title is not at the top of the panel:\n%s", text(f))
	}
	if st := styleAt(f, panelX, 0); st.FG != "#ffaf00" || !st.Bold {
		t.Errorf("the title's look: %+v", st)
	}
	next, score := -1, -1
	for i, r := range rows {
		if strings.Contains(r, "NEXT") && next < 0 {
			next = i
		}
		if strings.Contains(r, "SCORE") && score < 0 {
			score = i
		}
	}
	if next <= 1 || score <= next {
		t.Errorf("NEXT at row %d, SCORE at %d: the title first, then the next piece, then the scores", next, score)
	}
}

// TestTheFrameIsTheBoardThePieceItsGhostAndThePanel: the walls and floor, the falling piece in its
// colour, its ghost at the bottom, the next piece and the counts.
func TestTheFrameIsTheBoardThePieceItsGhostAndThePanel(t *testing.T) {
	g := game.New(7, game.MinHeight)
	g.Cur = game.Piece{Kind: game.T, X: 3, Y: game.Hidden}
	g.Score, g.Level, g.Lines = 1234, 3, 21
	f := render(g, needW, needH, dark, 5000, 0)
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
	g := game.New(1, game.MinHeight)
	if s := text(render(g, 30, 10, dark, 0, 0)); !strings.Contains(s, "Tetris needs a window") {
		t.Errorf("a small window:\n%s", s)
	}
	g.TogglePause()
	if s := text(render(g, needW, needH, dark, 0, 0)); !strings.Contains(s, "PAUSED") ||
		!strings.Contains(s, "p  resume") || !strings.Contains(s, "n  new game") || !strings.Contains(s, "q  quit") {
		t.Errorf("paused, the game's menu:\n%s", s)
	}
	g.TogglePause()
	g.Over = true
	if s := text(render(g, needW, needH, dark, 0, 0)); !strings.Contains(s, "GAME OVER") || !strings.Contains(s, "n  new game") {
		t.Errorf("over:\n%s", s)
	}
}

// TestTheWellFillsATallDialog: in a dialog taller than the guideline's board (80% of the screen),
// the well is as tall as the dialog, so the pieces fall the whole way; the panel sits at its top;
// a game taller than its window says so.
func TestTheWellFillsATallDialog(t *testing.T) {
	h := needH + 10
	g := game.New(1, wells(h))
	rows := strings.Split(text(render(g, needW, h, dark, 0, 0)), "\n")
	floor := -1
	for i, r := range rows {
		if strings.Contains(r, "└──") {
			floor = i
		}
	}
	if floor != h-1 || !strings.Contains(rows[3], "NEXT") || g.Height != h-1 {
		t.Fatalf("the floor at row %d (want %d), the well %d rows:\n%s", floor, h-1, g.Height, strings.Join(rows, "\n"))
	}
	if s := text(render(g, needW, needH, dark, 0, 0)); !strings.Contains(s, fmt.Sprintf("%d×%d", needW, h)) || !strings.Contains(s, "paused") {
		t.Errorf("a %d-row game in a %d-row window:\n%s", g.Height, needH, s)
	}
}

// TestTheKeysPlayTheGame: the arrows and Vim's keys move and turn, Space drops, p pauses, q quits,
// Enter restarts only a lost game, and a key that changes nothing says so.
func TestTheKeysPlayTheGame(t *testing.T) {
	g := game.New(3, game.MinHeight)
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
		{"q", changed, func() bool { return g.Paused }}, // q in play is the menu, not the end
		{"x", nothing, func() bool { return g.Paused }},
		{"q", quit, func() bool { return true }}, // the menu's quit
	}
	for i, s := range steps {
		if got := apply(g, plugin.Key{Key: s.key}); got != s.want || !s.ok() {
			t.Fatalf("step %d, %q: %v (want %v), state %+v", i, s.key, got, s.want, g.Cur)
		}
	}
	if apply(g, plugin.Key{Key: "n"}) != restart {
		t.Error("the menu's n does not start a new game")
	}
	g.Over = true
	if apply(g, plugin.Key{Key: "Enter"}) != restart || apply(g, plugin.Key{Key: "n"}) != restart ||
		apply(g, plugin.Key{Key: "q"}) != quit || apply(g, plugin.Key{Key: "Left"}) != nothing {
		t.Error("a lost game's menu: Enter and n again, q quits, the rest nothing")
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
	wait(func(s string) bool { return scoreOf(s) > 0 })           // a hard drop scores 2 a row
	_ = link.Notify(ctx, plugin.MethodHide, plugin.EmptyParams()) // Esc, in AutoDoc: the game pauses
	wait(func(s string) bool { return strings.Contains(s, "PAUSED") && strings.Contains(s, "q  quit") })
	_ = link.Notify(ctx, plugin.MethodShow, plugin.EmptyParams())
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

// fullRow is a game whose bottom row a vertical I has just filled: it is clearing.
func fullRow(t *testing.T) *game.Game {
	t.Helper()
	g := game.New(1, game.MinHeight)
	g.Board = make([][game.Width]game.Kind, game.MinHeight+game.Hidden)
	for x := 1; x < game.Width; x++ {
		g.Board[g.Rows()-1][x] = game.J
	}
	g.Cur = game.Piece{Kind: game.I, Rot: 1, X: -2, Y: 0}
	g.HardDrop()
	if len(g.Clearing) != 1 {
		t.Fatalf("the setup: Clearing %v", g.Clearing)
	}
	return g
}

// TestAClearIsSeenBlinkingThenWipedFromTheMiddle: the full row flashes bright white, then its own
// colours, then empties from the middle out, under a label of what it makes; the ghost and the
// next piece wait.
func TestAClearIsSeenBlinkingThenWipedFromTheMiddle(t *testing.T) {
	g := fullRow(t)
	floor := game.MinHeight - 1 // the bottom row, on the frame
	cell := func(f *plugin.Frame, x int) plugin.Style { return styleAt(f, left+1+x*2, floor) }
	flash := render(g, needW, needH, dark, 0, 0)
	if st := cell(flash, 5); st.FG != "brightwhite" || !st.Bold {
		t.Fatalf("step 0 is not the flash: %+v", st)
	}
	if !strings.Contains(text(flash), "SINGLE +100") || strings.Contains(text(flash), "░░") {
		t.Fatalf("the label, and no ghost while clearing:\n%s", text(flash))
	}
	if st := cell(render(g, needW, needH, dark, 0, 1), 5); st.FG != "blue" {
		t.Fatalf("step 1 is not the row's own colour: %+v", st)
	}
	wipe := render(g, needW, needH, dark, 0, clearBlinks) // the first wipe step: the middle two gone
	row := strings.Split(text(wipe), "\n")[floor]
	if got := []rune(row)[left+1+4*2]; got != ' ' {
		t.Fatalf("the middle is not wiped first: %q", row)
	}
	if got := []rune(row)[left+1]; got != '█' {
		t.Fatalf("the edge went before the middle: %q", row)
	}
	last := strings.Split(text(render(g, needW, needH, dark, 0, clearSteps-1)), "\n")[floor]
	if strings.Count(last, "█") != 0 {
		t.Fatalf("the last step leaves blocks: %q", last)
	}
}

// TestTheAnimationPlaysThenTheRowsGo: a lock that fills a row plays the animation, a frame each
// step, then takes the row away and the game plays on; Close stops it.
func TestTheAnimationPlaysThenTheRowsGo(t *testing.T) {
	var mu sync.Mutex
	var frames []string
	tt := &tetris{g: fullRow(t), w: needW, h: needH, theme: dark, stop: make(chan struct{})}
	tt.send = func(f *plugin.Frame) error {
		mu.Lock()
		frames = append(frames, text(f))
		mu.Unlock()
		return nil
	}
	tt.mu.Lock()
	tt.drawLocked()
	tt.mu.Unlock()
	deadline := time.Now().Add(5 * time.Second)
	for {
		tt.mu.Lock()
		done := tt.g.Clearing == nil
		tt.mu.Unlock()
		if done {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the rows never went")
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	n := len(frames)
	mu.Unlock()
	if n != clearSteps+1 || tt.g.Lines != 1 || tt.animating {
		t.Fatalf("%d frames (want %d), lines %d, animating %v", n, clearSteps+1, tt.g.Lines, tt.animating)
	}
	tt.mu.Lock()
	moved := tt.g.Move(1)
	tt.mu.Unlock()
	if !moved {
		t.Fatal("the game does not play on after the clear")
	}

	stopped := &tetris{g: fullRow(t), w: needW, h: needH, theme: dark, stop: make(chan struct{})}
	stopped.send = func(*plugin.Frame) error { return nil }
	stopped.mu.Lock()
	stopped.drawLocked()
	stopped.mu.Unlock()
	close(stopped.stop)
	time.Sleep(3 * clearStep)
	stopped.mu.Lock()
	defer stopped.mu.Unlock()
	if stopped.g.Clearing == nil {
		t.Fatal("the animation ran on after Close")
	}
}
