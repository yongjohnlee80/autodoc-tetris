package main

import (
	"fmt"

	"github.com/yongjohnlee80/autodoc/plugin"

	"github.com/yongjohnlee80/autodoc-tetris/game"
)

// The dialog: the board, two columns a cell, between its walls, and the panel beside it.
const (
	boardW = game.Width*2 + 2 // the cells and the two walls
	panelX = boardW + 2
	needW  = panelX + 18
	needH  = game.Height + 1 // the rows and the floor
)

// pieceColors are each kind's colour, in the themes' vocabulary: the guideline's, where a theme has
// colour. The mono theme has none: every piece is its brightest white, and the ghost tells them
// apart from the floor.
var pieceColors = map[game.Kind]string{
	game.I: "cyan", game.O: "yellow", game.T: "magenta", game.S: "green",
	game.Z: "red", game.J: "blue", game.L: "#ff8700",
}

// palette is how a theme draws the game.
type palette struct {
	piece map[game.Kind]plugin.Style
	wall  plugin.Style
	ghost plugin.Style
	text  plugin.Style
	title plugin.Style
}

func paletteFor(t plugin.Theme) palette {
	dim := t.Colors["document.lineNumber"]
	p := palette{piece: map[game.Kind]plugin.Style{}, wall: plugin.Style{FG: dim}, ghost: plugin.Style{FG: dim},
		title: plugin.Style{Bold: true}}
	for k, c := range pieceColors {
		if t.Name == "mono" {
			c = "brightwhite"
		}
		p.piece[k] = plugin.Style{FG: c}
	}
	return p
}

// render is the game, drawn into a w×h dialog.
func render(g *game.Game, w, h int, t plugin.Theme, best int) *plugin.Frame {
	f := plugin.NewFrame(w, h)
	pal := paletteFor(t)
	if w < needW || h < needH {
		f.Text(0, 0, "Tetris needs a window", pal.text)
		f.Text(0, 1, fmt.Sprintf("%d×%d; this one is %d×%d.", needW, needH, w, h), pal.text)
		f.Text(0, 3, "Esc closes.", pal.text)
		return f
	}
	for y := range game.Height {
		f.Set(0, y, '│', pal.wall)
		f.Set(boardW-1, y, '│', pal.wall)
		for x := range game.Width {
			if k := g.Board[y+game.Hidden][x]; k != 0 {
				block(f, x, y, "██", pal.piece[k])
			}
		}
	}
	f.Text(0, game.Height, "└"+repeat('─', boardW-2)+"┘", pal.wall)
	if !g.Over {
		for _, c := range g.Ghost().Cells() {
			if y := c[1] - game.Hidden; y >= 0 {
				block(f, c[0], y, "░░", pal.ghost)
			}
		}
		for _, c := range g.Cur.Cells() {
			if y := c[1] - game.Hidden; y >= 0 {
				block(f, c[0], y, "██", pal.piece[g.Cur.Kind])
			}
		}
	}

	f.Text(panelX, 0, "NEXT", pal.title)
	next := game.Piece{Kind: g.Next}
	for _, c := range next.Cells() {
		f.Text(panelX+c[0]*2, 1+c[1], "██", pal.piece[g.Next])
	}
	for i, row := range [][2]string{
		{"SCORE", fmt.Sprint(g.Score)}, {"LEVEL", fmt.Sprint(g.Level)},
		{"LINES", fmt.Sprint(g.Lines)}, {"BEST", fmt.Sprint(max(best, g.Score))},
	} {
		f.Text(panelX, 4+i*2, row[0], pal.title)
		f.Text(panelX, 5+i*2, row[1], pal.text)
	}
	for i, k := range []string{"←→   move", "↑ x  rotate", "z    rotate back", "↓    soft drop",
		"Spc  hard drop", "p    pause", "q    quit"} {
		f.Text(panelX, 13+i, k, pal.wall)
	}
	switch {
	case g.Over:
		centre(f, 8, " GAME OVER ", pal.title)
		centre(f, 10, " Enter: again ", pal.text)
	case g.Paused:
		centre(f, 9, " PAUSED ", pal.title)
	}
	return f
}

// block draws board cell (x, y) as two characters.
func block(f *plugin.Frame, x, y int, s string, st plugin.Style) { f.Text(1+x*2, y, s, st) }

// centre writes s centred over the board, on row y.
func centre(f *plugin.Frame, y int, s string, st plugin.Style) {
	n := len([]rune(s))
	f.Text((boardW-n)/2, y, s, st)
}

func repeat(r rune, n int) string {
	out := make([]rune, n)
	for i := range out {
		out[i] = r
	}
	return string(out)
}
