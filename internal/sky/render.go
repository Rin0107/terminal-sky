package sky

import (
	"math"
	"strings"
)

// Symbol は等級に応じた星の文字。明るいほど目立つ: ● > * > +。
func Symbol(mag float64) rune {
	switch {
	case mag < 0.5:
		return '●'
	case mag < 1.5:
		return '*'
	default:
		return '+'
	}
}

// RuneWidth は端末での文字の幅 (全角は 2、それ以外は 1)。
func RuneWidth(r rune) int {
	switch {
	case r >= 0x1100 && r <= 0x115F,
		r >= 0x2E80 && r <= 0xA4CF,
		r >= 0xAC00 && r <= 0xD7A3,
		r >= 0xF900 && r <= 0xFAFF,
		r >= 0xFE30 && r <= 0xFE4F,
		r >= 0xFF00 && r <= 0xFF60,
		r >= 0xFFE0 && r <= 0xFFE6:
		return 2
	default:
		return 1
	}
}

// Width は文字列の端末での幅。
func Width(s string) int {
	n := 0
	for _, r := range s {
		n += RuneWidth(r)
	}
	return n
}

// Truncate は幅 w に収まるところで文字列を切る。
func Truncate(s string, w int) string {
	n := 0
	for i, r := range s {
		if n+RuneWidth(r) > w {
			return s[:i]
		}
		n += RuneWidth(r)
	}
	return s
}

type kind int

const (
	empty   kind = iota
	horizon      // 地平線の点 (ラベルで上書きしてよい)
	solid        // 星・方角・ラベル (上書きしない)
	cont         // 全角の文字の右半分
)

type cell struct {
	r rune
	k kind
}

type grid struct {
	w, h  int
	cells [][]cell
}

func newGrid(w, h int) *grid {
	g := &grid{w: w, h: h, cells: make([][]cell, h)}
	for i := range g.cells {
		g.cells[i] = make([]cell, w)
		for j := range g.cells[i] {
			g.cells[i][j] = cell{r: ' '}
		}
	}
	return g
}

// free は (row, col) から幅 n が空いていて、両隣も星・ラベルでないか。
func (g *grid) free(row, col, n int) bool {
	if row < 0 || row >= g.h || col < 0 || col+n > g.w {
		return false
	}
	for c := col - 1; c <= col+n; c++ {
		if c < 0 || c >= g.w {
			continue
		}
		if k := g.cells[row][c].k; k == solid || k == cont {
			return false
		}
	}
	return true
}

func (g *grid) put(row, col int, s string) {
	for _, r := range s {
		g.cells[row][col] = cell{r: r, k: solid}
		if RuneWidth(r) == 2 {
			g.cells[row][col+1] = cell{k: cont}
			col++
		}
		col++
	}
}

func (g *grid) lines() []string {
	out := make([]string, g.h)
	for i, row := range g.cells {
		var b strings.Builder
		for _, c := range row {
			if c.k != cont {
				b.WriteRune(c.r)
			}
		}
		out[i] = strings.TrimRight(b.String(), " ")
	}
	return out
}

// Chart は空を真上から見た図 (天頂が中心、地平線が縁。上が北・左が東・下が南・右が西) を、
// 幅 w・高さ h の文字の格子に描く。seen は明るい順に並んでいること (先の星ほど優先して描く)。
// どの行も幅 w 以下で、行数は h 以下。
func Chart(seen []Seen, w, h int) []string {
	g := newGrid(w, h)
	rx, ry, cx, cy := layout(w, h)

	// 地平線
	for i := 0; i < 360; i++ {
		a := float64(i) * deg
		row := cy + int(math.Round(float64(ry)*math.Sin(a)))
		col := cx + int(math.Round(float64(rx)*math.Cos(a)))
		g.cells[row][col] = cell{r: '.', k: horizon}
	}
	g.put(cy-ry, cx, "N")
	g.put(cy+ry, cx, "S")
	g.put(cy, cx-rx, "E")
	g.put(cy, cx+rx, "W")

	// 星 (同じ升に重なったら明るい方を残す)
	type spot struct{ row, col int }
	spots := make([]spot, len(seen))
	drawn := make([]bool, len(seen))
	for i, s := range seen {
		row, col := project(s.Alt, s.Az, rx, ry, cx, cy)
		spots[i] = spot{row, col}
		if g.cells[row][col].k != solid {
			g.cells[row][col] = cell{r: Symbol(s.Star.Mag), k: solid}
			drawn[i] = true
		}
	}

	// 名前 (明るい順に、空いているところへ。右・左・右上・右下の順に試す)
	for i, s := range seen {
		if drawn[i] {
			g.label(spots[i].row, spots[i].col, s.Star.Name)
		}
	}
	return g.lines()
}

// layout は幅 w・高さ h の格子での、地平線の楕円の半径 (列 rx・行 ry) と中心 (cx, cy)。
// 端末の文字は縦長なので、列の半径は行の半径の 2 倍にする (幅が足りなければ幅に合わせる)。
// 仕様: spec/sky.smt2 の 1 (楕円が格子に収まる)。
func layout(w, h int) (rx, ry, cx, cy int) {
	ry = (h - 1) / 2
	rx = 2 * ry
	if lim := (w - 1) / 2; rx > lim {
		rx = lim
		ry = rx / 2
	}
	return rx, ry, (w - 1) / 2, ry
}

// project は高度 alt・方位 az の星を格子の (行, 列) に置く。天頂が中心、地平線が楕円の縁。
// 仕様: spec/sky.smt2 の 3 (高度 0〜90 の星は楕円の中に入る)。
func project(alt, az float64, rx, ry, cx, cy int) (row, col int) {
	r := (90 - alt) / 90
	a := az * deg
	row = cy + int(math.Round(-math.Cos(a)*r*float64(ry)))
	col = cx + int(math.Round(-math.Sin(a)*r*float64(rx)))
	return row, col
}

// label は (row, col) の星の名前を、右・左・右上・右下の順に空いているところへ置く。置けなければ置かない。
func (g *grid) label(row, col int, name string) {
	n := Width(name)
	for _, c := range [][2]int{{row, col + 2}, {row, col - 1 - n}, {row - 1, col + 1}, {row + 1, col + 1}} {
		if g.free(c[0], c[1], n) {
			g.put(c[0], c[1], name)
			return
		}
	}
}
