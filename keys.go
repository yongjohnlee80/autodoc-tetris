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
// anticlockwise; Space drops; p pauses; q quits; Enter starts a lost game again.
func apply(g *game.Game, k plugin.Key) action {
	switch k.Key {
	case "q", "Q":
		return quit
	case "Enter":
		if g.Over {
			return restart
		}
		return nothing
	case "p", "P":
		g.TogglePause()
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
	if g.Cur == before.Cur && g.Score == before.Score && g.Board == before.Board {
		return nothing
	}
	return changed
}
