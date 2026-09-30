package main

import (
	"github.com/yongjohnlee80/autodoc/plugin"

	"github.com/yongjohnlee80/autodoc-tetris/game"
)

// action is what a key does: the game changed, or it asks to quit, or to start again.
type action int

const (
	nothing action = iota
	changed
	quit
	restart
)

// apply plays key k on g. The arrows, and Vim's h j k l, move; up, x and k turn clockwise, z
// anticlockwise; Space drops. p, or q, pauses and opens the game's menu; in the menu p resumes, n
// starts a new game and q quits, and a lost game's menu is the same with Enter for a new one. Esc
// is AutoDoc's: it hides the dialog, and the game pauses (tetris.Hide).
func apply(g *game.Game, k plugin.Key) action {
	switch {
	case g.Over:
		switch k.Key {
		case "Enter", "n", "N":
			return restart
		case "q", "Q":
			return quit
		}
		return nothing
	case g.Paused:
		switch k.Key {
		case "p", "P":
			g.TogglePause()
			return changed
		case "n", "N":
			return restart
		case "q", "Q":
			return quit
		}
		return nothing
	}
	switch k.Key {
	case "p", "P", "q", "Q":
		g.TogglePause() // the menu, not the end: quitting is the menu's
		return changed
	}
	before := *g
	switch k.Key {
	case "Left", "h":
		g.Move(-1)
	case "Right", "l":
		g.Move(1)
	case "Down", "j":
		g.SoftDrop()
	case "Up", "x", "X", "k":
		g.Rotate(1)
	case "z", "Z":
		g.Rotate(-1)
	case " ":
		g.HardDrop()
	default:
		return nothing
	}
	// a lock spawns the next piece, so a changed board is a changed Cur
	if g.Cur == before.Cur && g.Score == before.Score {
		return nothing
	}
	return changed
}
