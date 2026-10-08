package main

import (
	"fmt"
	"time"

	"github.com/yongjohnlee80/autodoc/plugin"

	"github.com/yongjohnlee80/autodoc-tetris/game"
)

// The dialog: a column of margin, the board, two columns a cell, between its walls, and the panel
// beside it. The margin keeps the board's wall off the dialog's border.
const (
	left   = 1
	boardW = game.Width*2 + 2 // the cells and the two walls
	panelX = left + boardW + 2
	needW  = panelX + 18
	needH  = game.MinHeight + 1 // the guideline's rows and the floor: a game in a taller window is taller
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
	name  plugin.Style // the title: the theme's accent (its cursor's colour), bold
}

// titleLines are the game's title, at the top of the score panel.
var titleLines = []string{"AutoTetris"}

func paletteFor(t plugin.Theme) palette {
	dim := t.Colors["document.lineNumber"]
	p := palette{piece: map[game.Kind]plugin.Style{}, wall: plugin.Style{FG: dim}, ghost: plugin.Style{FG: dim},
		title: plugin.Style{Bold: true}, name: plugin.Style{FG: t.Colors["document.cursor"], Bold: true}}
	for k, c := range pieceColors {
		if t.Name == "mono" {
			c = "brightwhite"
		}
		p.piece[k] = plugin.Style{FG: c}
	}
	return p
}

// render is the game, drawn into a w×h dialog. The well is the game's height (wells returns it for
// a dialog), so in a tall dialog — the manifest asks for 80% of the screen — the pieces fall the
// whole way down; the panel sits beside its top.
func render(g *game.Game, w, h int, t plugin.Theme, best int, step int) *plugin.Frame {
	f := plugin.NewFrame(w, h)
	pal := paletteFor(t)
	if need := g.Height + 1; w < needW || h < need {
		f.Text(0, 0, "AutoTetris needs a window", pal.text)
		f.Text(0, 1, fmt.Sprintf("%d×%d; this one is %d×%d.", needW, need, w, h), pal.text)
		f.Text(0, 3, "The game is paused. Esc hides it.", pal.text)
		return f
	}
	top := (h - g.Height - 1) / 2 // 0 when the well fills the dialog
	block := func(x, y int, s string, st plugin.Style) { f.Text(left+1+x*2, top+y, s, st) }
	centre := func(y int, s string, st plugin.Style) {
		f.Text(left+(boardW-len([]rune(s)))/2, top+y, s, st)
	}
	clearing := map[int]bool{}
	for _, y := range g.Clearing {
		clearing[y-game.Hidden] = true
	}
	for y := range g.Height {
		f.Set(left, top+y, '│', pal.wall)
		f.Set(left+boardW-1, top+y, '│', pal.wall)
		for x := range game.Width {
			k := g.Board[y+game.Hidden][x]
			switch {
			case k == 0:
			case clearing[y]:
				if st, ok := clearLook(step, x, pal.piece[k]); ok {
					block(x, y, "██", st)
				}
			default:
				block(x, y, "██", pal.piece[k])
			}
		}
	}
	f.Text(left, top+g.Height, "└"+repeat('─', boardW-2)+"┘", pal.wall)
	if g.Clearing != nil {
		// what the rows make, over the board above them
		name := [...]string{"", "SINGLE", "DOUBLE", "TRIPLE", "TETRIS"}[min(len(g.Clearing), 4)]
		above := g.Clearing[len(g.Clearing)-1] - game.Hidden - 2 // the topmost full row, bottom first
		centre(max(above, 0), fmt.Sprintf(" %s +%d ", name, g.ClearPoints()), pal.title)
	} else if !g.Over {
		for _, c := range g.Ghost().Cells() {
			if y := c[1] - game.Hidden; y >= 0 {
				block(c[0], y, "░░", pal.ghost)
			}
		}
		for _, c := range g.Cur.Cells() {
			if y := c[1] - game.Hidden; y >= 0 {
				block(c[0], y, "██", pal.piece[g.Cur.Kind])
			}
		}
	}

	// the title, at the top of the panel
	for i, line := range titleLines {
		f.Text(panelX, top+i, line, pal.name)
	}
	f.Text(panelX, top+3, "NEXT", pal.title)
	next := game.Piece{Kind: g.Next}
	for _, c := range next.Cells() {
		f.Text(panelX+c[0]*2, top+4+c[1], "██", pal.piece[g.Next])
	}
	for i, row := range [][2]string{
		{"SCORE", fmt.Sprint(g.Score)}, {"LEVEL", fmt.Sprint(g.Level)},
		{"LINES", fmt.Sprint(g.Lines)}, {"BEST", fmt.Sprint(max(best, g.Score))},
	} {
		f.Text(panelX, top+7+i, row[0], pal.title)
		f.Text(panelX+7, top+7+i, row[1], pal.text)
	}
	for i, k := range []string{"←→   move", "↑ x  rotate", "z    rotate back", "↓    soft drop",
		"Spc  hard drop", "p q  menu", "Esc  hide"} {
		f.Text(panelX, top+12+i, k, pal.wall)
	}
	// the game's menu: over the board while paused, and when lost
	switch {
	case g.Over:
		centre(7, " GAME OVER ", pal.title)
		centre(9, " n  new game ", pal.text)
		centre(10, " q  quit     ", pal.text)
	case g.Paused:
		centre(7, " PAUSED ", pal.title)
		centre(9, " p  resume   ", pal.text)
		centre(10, " n  new game ", pal.text)
		centre(11, " q  quit     ", pal.text)
	}
	return f
}

// The clearing animation: clearBlinks steps flashing the full rows (bright white, then their own
// colours), then clearWipe steps emptying them from the middle outwards, clearStep apart.
const (
	clearBlinks = 4
	clearWipe   = game.Width / 2
	clearSteps  = clearBlinks + clearWipe
	clearStep   = 55 * time.Millisecond
)

// clearLook is a full row's cell x at step of the animation: its look, or false when the wipe has
// reached it.
func clearLook(step, x int, own plugin.Style) (plugin.Style, bool) {
	if step < clearBlinks {
		if step%2 == 0 {
			return plugin.Style{FG: "brightwhite", Bold: true}, true
		}
		return own, true
	}
	gone := step - clearBlinks + 1 // columns gone each side of the middle
	if x >= game.Width/2-gone && x < game.Width/2+gone {
		return plugin.Style{}, false
	}
	return own, true
}

// wells is the well's height for a dialog h rows tall: all of it but the floor.
func wells(h int) int { return h - 1 }

func repeat(r rune, n int) string {
	out := make([]rune, n)
	for i := range out {
		out[i] = r
	}
	return string(out)
}
