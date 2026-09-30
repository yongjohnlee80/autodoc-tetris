package game

import (
	"slices"
	"testing"
	"time"
)

func cellsOf(p Piece) [][2]int {
	c := p.Cells()
	out := c[:]
	slices.SortFunc(out, func(a, b [2]int) int {
		if a[1] != b[1] {
			return a[1] - b[1]
		}
		return a[0] - b[0]
	})
	return out
}

// empty is a game on an empty board with piece k, rotation 0, at its spawn.
func empty(k Kind) *Game {
	g := New(1)
	g.Board = [Rows][Width]Kind{}
	g.Cur = Piece{Kind: k, X: 3}
	return g
}

// TestTheBagDealsEachPieceOnceInSeven: every seven pieces are the seven kinds.
func TestTheBagDealsEachPieceOnceInSeven(t *testing.T) {
	g := New(42)
	g.bag = nil // New dealt two: start at a bag's edge
	for bag := range 4 {
		seen := map[Kind]int{}
		for range 7 {
			seen[g.draw()]++
		}
		for _, k := range Kinds {
			if seen[k] != 1 {
				t.Fatalf("bag %d: %v dealt %d times (%v)", bag, k, seen[k], seen)
			}
		}
	}
}

// TestPiecesSpawnWhereTheGuidelineSays: O in columns 4–5, I in 3–6, the rest in 3–5, in the hidden
// rows.
func TestPiecesSpawnWhereTheGuidelineSays(t *testing.T) {
	for k, want := range map[Kind][][2]int{
		O: {{4, 0}, {5, 0}, {4, 1}, {5, 1}},
		I: {{3, 1}, {4, 1}, {5, 1}, {6, 1}},
		T: {{4, 0}, {3, 1}, {4, 1}, {5, 1}},
	} {
		if got := cellsOf(Piece{Kind: k, X: 3}); !slices.Equal(got, want) {
			t.Errorf("%v spawns at %v, want %v", k, got, want)
		}
	}
}

// TestRotationTurnsTheBoxClockwise: T's four states, and O's one.
func TestRotationTurnsTheBoxClockwise(t *testing.T) {
	g := empty(T)
	g.Cur.Y = 5
	want := [][][2]int{
		{{4, 5}, {3, 6}, {4, 6}, {5, 6}}, // up
		{{4, 5}, {4, 6}, {5, 6}, {4, 7}}, // right
		{{3, 6}, {4, 6}, {5, 6}, {4, 7}}, // down
		{{4, 5}, {3, 6}, {4, 6}, {4, 7}}, // left
	}
	for i := range 4 {
		if got := cellsOf(g.Cur); !slices.Equal(got, want[i]) {
			t.Fatalf("T rotation %d: %v, want %v", i, got, want[i])
		}
		if !g.Rotate(1) {
			t.Fatalf("T did not rotate from %d in open space", i)
		}
	}
	if g.Cur.Rot != 0 {
		t.Errorf("four turns ended at rotation %d", g.Cur.Rot)
	}
	g.Rotate(-1)
	if g.Cur.Rot != 3 {
		t.Errorf("a counter-clockwise turn from 0 is rotation %d, want 3 (L)", g.Cur.Rot)
	}
	o := empty(O)
	if o.Rotate(1) {
		t.Error("O rotated")
	}
}

// TestAWallKickMovesThePieceOffTheWall: T standing on its left against the left wall turns by
// kicking right, as SRS's L→0 table first tries (+1, 0); I kicks by its own table.
func TestAWallKickMovesThePieceOffTheWall(t *testing.T) {
	g := empty(T)
	g.Cur = Piece{Kind: T, Rot: 1, X: -1, Y: 5} // R: its column 0 empty, so the box hangs off
	if !g.fits(g.Cur) {
		t.Fatal("the setup does not fit")
	}
	if !g.Rotate(1) {
		t.Fatal("T did not rotate against the wall")
	}
	for _, c := range g.Cur.Cells() {
		if c[0] < 0 {
			t.Fatalf("the kick left T in the wall: %v", g.Cur.Cells())
		}
	}
	i := empty(I)
	i.Cur = Piece{Kind: I, Rot: 1, X: -2, Y: 5} // vertical in column 0
	if !i.Rotate(1) || cellsOf(i.Cur)[0][0] != 0 {
		t.Fatalf("I against the left wall: %v, %v", i.Cur, i.Cur.Cells())
	}
}

// TestMovesStopAtTheWallsAndTheStack: a piece moves until a wall or a filled cell is in the way.
func TestMovesStopAtTheWallsAndTheStack(t *testing.T) {
	g := empty(O)
	n := 0
	for g.Move(-1) {
		n++
	}
	if n != 4 {
		t.Errorf("O moved %d left from its spawn, want 4", n)
	}
	g.Board[1][7] = I
	g.Cur = Piece{Kind: O, X: 3}
	n = 0
	for g.Move(1) {
		n++
	}
	if n != 1 {
		t.Errorf("O moved %d right towards a filled cell two away, want 1", n)
	}
}

// TestLinesClearAndScoreByTheGuideline: a full row goes and what is above falls; one line is 100
// times the level, four at once 800; ten lines make the next level.
func TestLinesClearAndScoreByTheGuideline(t *testing.T) {
	g := empty(I)
	for x := 1; x < Width; x++ {
		for y := Rows - 4; y < Rows; y++ {
			g.Board[y][x] = J
		}
	}
	g.Board[Rows-5][5] = S // above the four rows: falls four
	g.Cur = Piece{Kind: I, Rot: 1, X: -2, Y: 0}
	n := g.HardDrop()
	if g.Lines != 4 || g.Score != 800+2*n {
		t.Fatalf("a tetris: %d lines, score %d, want 4 and %d", g.Lines, g.Score, 800+2*n)
	}
	if g.Board[Rows-1][5] != S {
		t.Errorf("the cell above the cleared rows did not fall: %v", g.Board[Rows-1])
	}
	for y := range Rows - 1 {
		for x := range Width {
			if g.Board[y][x] != 0 {
				t.Fatalf("row %d not empty after the clear: %v", y, g.Board[y])
			}
		}
	}

	g = empty(I)
	g.Lines, g.Level = 9, 1
	for x := 4; x < Width; x++ {
		g.Board[Rows-1][x] = Z
	}
	g.Cur = Piece{Kind: I, X: 0, Y: 0}
	fell := g.HardDrop()
	if g.Lines != 10 || g.Level != 2 || g.Score != 100+2*fell { // scored at the level it was cleared at
		t.Fatalf("the tenth line: lines %d level %d score %d", g.Lines, g.Level, g.Score)
	}
}

// TestDropsScoreAndTheGhostIsWhereItLands: soft drop scores 1 a row, hard drop 2; the ghost is the
// hard drop's landing.
func TestDropsScoreAndTheGhostIsWhereItLands(t *testing.T) {
	g := empty(T)
	g.SoftDrop()
	g.SoftDrop()
	if g.Score != 2 || g.Cur.Y != 2 {
		t.Fatalf("two soft drops: score %d, y %d", g.Score, g.Cur.Y)
	}
	ghost := g.Ghost()
	landed := cellsOf(ghost)
	fell := g.HardDrop()
	if g.Score != 2+2*fell || fell != ghost.Y-2 {
		t.Fatalf("hard drop fell %d (ghost %d), score %d", fell, ghost.Y, g.Score)
	}
	for _, c := range landed {
		if g.Board[c[1]][c[0]] != T {
			t.Fatalf("T did not lock where its ghost was: %v", landed)
		}
	}
}

// TestGravityLocksAndTheGameEndsWhenAPieceCannotSpawn: ticks move the piece down and lock it at the
// bottom; a blocked spawn is the game over, after which nothing moves.
func TestGravityLocksAndTheGameEndsWhenAPieceCannotSpawn(t *testing.T) {
	g := empty(O)
	for range Rows {
		g.Tick()
	}
	if g.Board[Rows-1][4] != O {
		t.Fatal("gravity did not lock O at the bottom")
	}
	for x := range Width {
		g.Board[1][x] = L
	}
	g.Board[1][0] = 0 // not a full row
	g.Cur = Piece{Kind: O, X: 3}
	g.HardDrop()
	if !g.Over {
		t.Fatal("a blocked spawn did not end the game")
	}
	score := g.Score
	g.HardDrop()
	g.TogglePause()
	if g.Score != score || g.Paused {
		t.Error("a game over still plays")
	}
}

// TestPauseStopsEverything: paused, no move, drop or tick does anything.
func TestPauseStopsEverything(t *testing.T) {
	g := empty(T)
	g.TogglePause()
	before := g.Cur
	g.Move(1)
	g.Rotate(1)
	g.SoftDrop()
	g.Tick()
	if g.HardDrop() != 0 || g.Cur != before {
		t.Fatal("a paused game moved")
	}
	g.TogglePause()
	if !g.Move(1) {
		t.Fatal("resumed, the game does not move")
	}
}

// TestGravityQuickensByLevel: level 1 is a second a row, level 2 0.793 s, each level faster, never
// under 30 ms.
func TestGravityQuickensByLevel(t *testing.T) {
	if g := Gravity(2); g != 793*time.Millisecond {
		t.Errorf("level 2: %v", g)
	}
	if g := Gravity(1); g != time.Second {
		t.Errorf("level 1: %v", g)
	}
	for l := 2; l <= 30; l++ {
		if Gravity(l) > Gravity(l-1) || Gravity(l) < 30*time.Millisecond {
			t.Fatalf("level %d: %v after %v", l, Gravity(l), Gravity(l-1))
		}
	}
	if Gravity(0) != Gravity(1) {
		t.Error("level 0 is not level 1")
	}
	if I.String() != "I" || L.String() != "L" {
		t.Error("Kind.String")
	}
}
