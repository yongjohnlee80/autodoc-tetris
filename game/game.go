// Package game is Tetris, as the guideline plays it, with nothing of the screen in it: a 10×20
// board with two rows above it that pieces spawn into, the seven tetrominoes from a 7-bag,
// Super Rotation System rotation with its wall kicks, gravity that quickens every level, and the
// guideline's scoring. The plugin draws it and drives it; a test drives it with a seed.
package game

import (
	"math"
	"math/rand"
	"time"
)

// The board: Width columns, a game's Height visible rows (the guideline's MinHeight at least,
// taller in a taller window, so the pieces fall further), and Hidden rows above them where a piece
// spawns.
const (
	Width     = 10
	MinHeight = 20
	Hidden    = 2
)

// Kind is a tetromino.
type Kind uint8

// The seven, in the guideline's order.
const (
	I Kind = iota + 1
	O
	T
	S
	Z
	J
	L
)

// Kinds are all seven.
var Kinds = []Kind{I, O, T, S, Z, J, L}

func (k Kind) String() string { return string("?IOTSZJL"[k]) }

// shapes are each kind's cells at rotation 0, in its box: I's 4×4, O's 2×2 (from its 4-wide spawn
// box's column 1), the rest 3×3. Rotation turns the box.
var shapes = map[Kind][][2]int{
	I: {{0, 1}, {1, 1}, {2, 1}, {3, 1}},
	O: {{1, 0}, {2, 0}, {1, 1}, {2, 1}},
	T: {{1, 0}, {0, 1}, {1, 1}, {2, 1}},
	S: {{1, 0}, {2, 0}, {0, 1}, {1, 1}},
	Z: {{0, 0}, {1, 0}, {1, 1}, {2, 1}},
	J: {{0, 0}, {0, 1}, {1, 1}, {2, 1}},
	L: {{2, 0}, {0, 1}, {1, 1}, {2, 1}},
}

// box is a kind's rotation box's side.
func box(k Kind) int {
	switch k {
	case I, O:
		return 4
	}
	return 3
}

// Piece is a tetromino on the board: its kind, its rotation (0, 1 = R, 2, 3 = L) and its box's
// top-left corner, in board cells, y growing downwards and row 0 the top hidden row.
type Piece struct {
	Kind Kind
	Rot  int
	X, Y int
}

// Cells are the board cells p covers.
func (p Piece) Cells() [4][2]int {
	var out [4][2]int
	n := box(p.Kind)
	for i, c := range shapes[p.Kind] {
		x, y := c[0], c[1]
		if p.Kind != O {
			for range p.Rot {
				x, y = n-1-y, x // a quarter turn clockwise in the box
			}
		}
		out[i] = [2]int{p.X + x, p.Y + y}
	}
	return out
}

// kicks are SRS's wall kicks, by rotation from→to, as (dx, dy) with y up, as the guideline writes
// them: tried in order, the first that fits wins.
var kicks = map[[2]int][5][2]int{
	{0, 1}: {{0, 0}, {-1, 0}, {-1, 1}, {0, -2}, {-1, -2}},
	{1, 0}: {{0, 0}, {1, 0}, {1, -1}, {0, 2}, {1, 2}},
	{1, 2}: {{0, 0}, {1, 0}, {1, -1}, {0, 2}, {1, 2}},
	{2, 1}: {{0, 0}, {-1, 0}, {-1, 1}, {0, -2}, {-1, -2}},
	{2, 3}: {{0, 0}, {1, 0}, {1, 1}, {0, -2}, {1, -2}},
	{3, 2}: {{0, 0}, {-1, 0}, {-1, -1}, {0, 2}, {-1, 2}},
	{3, 0}: {{0, 0}, {-1, 0}, {-1, -1}, {0, 2}, {-1, 2}},
	{0, 3}: {{0, 0}, {1, 0}, {1, 1}, {0, -2}, {1, -2}},
}

var kicksI = map[[2]int][5][2]int{
	{0, 1}: {{0, 0}, {-2, 0}, {1, 0}, {-2, -1}, {1, 2}},
	{1, 0}: {{0, 0}, {2, 0}, {-1, 0}, {2, 1}, {-1, -2}},
	{1, 2}: {{0, 0}, {-1, 0}, {2, 0}, {-1, 2}, {2, -1}},
	{2, 1}: {{0, 0}, {1, 0}, {-2, 0}, {1, -2}, {-2, 1}},
	{2, 3}: {{0, 0}, {2, 0}, {-1, 0}, {2, 1}, {-1, -2}},
	{3, 2}: {{0, 0}, {-2, 0}, {1, 0}, {-2, -1}, {1, 2}},
	{3, 0}: {{0, 0}, {1, 0}, {-2, 0}, {1, -2}, {-2, 1}},
	{0, 3}: {{0, 0}, {-1, 0}, {2, 0}, {-1, 2}, {2, -1}},
}

// Game is one game.
type Game struct {
	// Board is each cell's kind, 0 for empty; row 0 is the top hidden row, and it has Rows() rows.
	Board [][Width]Kind
	// Height is the visible rows: MinHeight or more.
	Height int
	Cur    Piece
	Next   Kind
	Score  int
	Lines  int
	Level  int
	// Over is the game lost: a piece could not spawn. Paused stops gravity and the keys but pause.
	Over, Paused bool

	rng *rand.Rand
	bag []Kind
}

// New is a game height rows tall (MinHeight at least), its pieces drawn from seed.
func New(seed int64, height int) *Game {
	height = max(height, MinHeight)
	g := &Game{Level: 1, Height: height, Board: make([][Width]Kind, height+Hidden), rng: rand.New(rand.NewSource(seed))}
	g.Next = g.draw()
	g.spawn()
	return g
}

// draw is the bag's next piece: each seven in a random order, then a new seven.
func (g *Game) draw() Kind {
	if len(g.bag) == 0 {
		g.bag = append(g.bag, Kinds...)
		g.rng.Shuffle(len(g.bag), func(i, j int) { g.bag[i], g.bag[j] = g.bag[j], g.bag[i] })
	}
	k := g.bag[0]
	g.bag = g.bag[1:]
	return k
}

// spawn puts Next at the top, centred, and draws the one after it; a spawn that does not fit is
// the game over.
func (g *Game) spawn() {
	p := Piece{Kind: g.Next, X: 3, Y: 0}
	g.Next = g.draw()
	if !g.fits(p) {
		g.Over = true
	}
	g.Cur = p
}

// fits says p is inside the board and over no filled cell.
func (g *Game) fits(p Piece) bool {
	for _, c := range p.Cells() {
		x, y := c[0], c[1]
		if x < 0 || x >= Width || y < 0 || y >= len(g.Board) || g.Board[y][x] != 0 {
			return false
		}
	}
	return true
}

func (g *Game) playing() bool { return !g.Over && !g.Paused }

// Move shifts the piece dx columns, when it fits.
func (g *Game) Move(dx int) bool {
	if !g.playing() {
		return false
	}
	p := g.Cur
	p.X += dx
	if !g.fits(p) {
		return false
	}
	g.Cur = p
	return true
}

// Rotate turns the piece a quarter, clockwise for dir > 0, trying SRS's kicks in order.
func (g *Game) Rotate(dir int) bool {
	if !g.playing() || g.Cur.Kind == O {
		return false
	}
	from := g.Cur.Rot
	to := (from + 4 + sign(dir)) % 4
	table := kicks
	if g.Cur.Kind == I {
		table = kicksI
	}
	for _, k := range table[[2]int{from, to}] {
		p := g.Cur
		p.Rot, p.X, p.Y = to, p.X+k[0], p.Y-k[1] // the table's y is up
		if g.fits(p) {
			g.Cur = p
			return true
		}
	}
	return false
}

func sign(n int) int {
	if n < 0 {
		return -1
	}
	return 1
}

// SoftDrop moves the piece down a row, scoring 1; at the bottom it locks.
func (g *Game) SoftDrop() {
	if !g.playing() {
		return
	}
	if g.down() {
		g.Score++
		return
	}
	g.lock()
}

// HardDrop drops the piece to the bottom and locks it, scoring 2 a row, and says how far it fell.
func (g *Game) HardDrop() int {
	if !g.playing() {
		return 0
	}
	n := 0
	for g.down() {
		n++
	}
	g.Score += 2 * n
	g.lock()
	return n
}

// Tick is gravity: the piece a row down, or locked where it lies.
func (g *Game) Tick() {
	if !g.playing() {
		return
	}
	if !g.down() {
		g.lock()
	}
}

func (g *Game) down() bool {
	p := g.Cur
	p.Y++
	if !g.fits(p) {
		return false
	}
	g.Cur = p
	return true
}

// Ghost is where the piece would land.
func (g *Game) Ghost() Piece {
	p := g.Cur
	for {
		q := p
		q.Y++
		if !g.fits(q) {
			return p
		}
		p = q
	}
}

// lineScores are the guideline's points for 1 to 4 lines at once, times the level.
var lineScores = [5]int{0, 100, 300, 500, 800}

// lock writes the piece into the board, clears the full rows, scores them, and spawns the next.
func (g *Game) lock() {
	for _, c := range g.Cur.Cells() {
		g.Board[c[1]][c[0]] = g.Cur.Kind
	}
	cleared := 0
	for y := len(g.Board) - 1; y >= 0; y-- {
		full := true
		for x := range Width {
			if g.Board[y][x] == 0 {
				full = false
				break
			}
		}
		if !full {
			continue
		}
		cleared++
		copy(g.Board[1:y+1], g.Board[0:y])
		g.Board[0] = [Width]Kind{}
		y++ // the row that moved into y is looked at again
	}
	g.Score += lineScores[cleared] * g.Level
	g.Lines += cleared
	g.Level = 1 + g.Lines/10
	g.spawn()
}

// Rows is the board's rows, the hidden ones with them.
func (g *Game) Rows() int { return len(g.Board) }

// Resize makes the visible rows height (MinHeight at least), at the top of the well, so the stack
// stays where it lies: growing adds empty rows above; shrinking drops rows from the top only while
// they are empty and the piece is not in them. It says whether the game is now height rows tall.
func (g *Game) Resize(height int) bool {
	height = max(height, MinHeight)
	for g.Height < height {
		g.Board = append([][Width]Kind{{}}, g.Board...)
		g.Height++
		g.Cur.Y++
	}
	for g.Height > height && g.Board[0] == ([Width]Kind{}) && g.Cur.Y > 0 {
		g.Board = g.Board[1:]
		g.Height--
		g.Cur.Y--
	}
	return g.Height == height
}

// TogglePause pauses or resumes the game; a game over stays over.
func (g *Game) TogglePause() {
	if !g.Over {
		g.Paused = !g.Paused
	}
}

// Gravity is a level's time between rows, the guideline's (0.8 − 0.007·(level−1))^(level−1)
// seconds, never under 30 ms.
func Gravity(level int) time.Duration {
	l := float64(max(level, 1) - 1)
	s := math.Pow(0.8-0.007*l, l)
	return max(time.Duration(s*float64(time.Second)), 30*time.Millisecond)
}
