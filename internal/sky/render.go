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
	line         // 星座線の点 (星・ラベルで上書きしてよい)
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

// Chart は明るい星だけの図 (星座なし)。Draw(Scene{Stars: seen}, w, h) と同じ。
func Chart(seen []Seen, w, h int) []string {
	return Draw(Scene{Stars: seen}, w, h)
}

// Scene は図に描くもの。
type Scene struct {
	Stars   []Seen   // 名前を付けて描く明るい星 (明るい順)
	Figures []Figure // 星座 (線・線の星の点・星座の名前)
	NoLines bool     // true なら星座線を描かない (星の点と星座の名前は描く)
}

// Draw は空を真上から見た図 (天頂が中心、地平線が縁。上が北・左が東・下が南・右が西) を、
// 幅 w・高さ h の文字の格子に描く。重なったら先に描いたものが残る:
// 方角 → 明るい星 (明るい順) → 1 等星の名前 → 星座の星の点 → 星座の名前 → 2 等星の名前。
// 狭い図では場所を取り合うので、1 等星 (● と *) の名前を一番にし、2 等星 (+) の名前は星座の名前に譲る。
// 星座線と地平線の点は、星と名前に場所を譲る。どの行も幅 w 以下で、行数は h 以下。
func Draw(sc Scene, w, h int) []string {
	g := newGrid(w, h)
	v := newView(w, h)
	g.horizon(v)
	if !sc.NoLines {
		for _, f := range sc.Figures {
			for _, s := range f.Segments {
				g.constellationLine(v, s)
			}
		}
	}
	drawn := g.stars(v, sc.Stars)
	g.starLabels(v, sc.Stars, drawn, true)
	for _, f := range sc.Figures {
		g.points(v, f.Stars)
	}
	for _, f := range sc.Figures {
		g.figureName(v, f)
	}
	g.starLabels(v, sc.Stars, drawn, false)
	return g.lines()
}

// view は格子の上の地平線の楕円 (半径 rx・ry、中心 cx・cy)。
type view struct{ rx, ry, cx, cy int }

func newView(w, h int) view {
	rx, ry, cx, cy := layout(w, h)
	return view{rx, ry, cx, cy}
}

// at は星を置く升 (行, 列)。
func (v view) at(s Seen) (row, col int) { return project(s.Alt, s.Az, v.rx, v.ry, v.cx, v.cy) }

// horizon は地平線の点と方角 (N・S・E・W) を描く。
func (g *grid) horizon(v view) {
	for i := 0; i < 360; i++ {
		a := float64(i) * deg
		row := v.cy + int(math.Round(float64(v.ry)*math.Sin(a)))
		col := v.cx + int(math.Round(float64(v.rx)*math.Cos(a)))
		g.cells[row][col] = cell{r: '.', k: horizon}
	}
	g.put(v.cy-v.ry, v.cx, "N")
	g.put(v.cy+v.ry, v.cx, "S")
	g.put(v.cy, v.cx-v.rx, "E")
	g.put(v.cy, v.cx+v.rx, "W")
}

// stars は明るい星を描き、描けた (もっと明るい星に先を越されなかった) かを返す。
func (g *grid) stars(v view, seen []Seen) []bool {
	drawn := make([]bool, len(seen))
	for i, s := range seen {
		row, col := v.at(s)
		if g.cells[row][col].k != solid {
			g.cells[row][col] = cell{r: Symbol(s.Star.Mag), k: solid}
			drawn[i] = true
		}
	}
	return drawn
}

// starLabels は描けた星の名前を明るい順に置く。first なら 1 等星 (● と *) だけ、でなければ 2 等星 (+) だけ。
func (g *grid) starLabels(v view, seen []Seen, drawn []bool, first bool) {
	for i, s := range seen {
		if drawn[i] && (Symbol(s.Star.Mag) != '+') == first {
			row, col := v.at(s)
			g.label(row, col, s.Star.Name)
		}
	}
}

// points は星座の星を点だけ (名前なしで) 描く。星・名前の升には置かない。
func (g *grid) points(v view, stars []Seen) {
	for _, s := range stars {
		row, col := v.at(s)
		if k := g.cells[row][col].k; k != solid && k != cont {
			g.cells[row][col] = cell{r: Symbol(s.Star.Mag), k: solid}
		}
	}
}

// figureName は星座の名前を、見えている星の真ん中に置く (入らなければ近くにずらす)。
func (g *grid) figureName(v view, f Figure) {
	sumR, sumC := 0, 0
	for _, s := range f.Stars {
		row, col := v.at(s)
		sumR, sumC = sumR+row, sumC+col
	}
	n := float64(len(f.Stars))
	g.center(int(math.Round(float64(sumR)/n)), int(math.Round(float64(sumC)/n)), f.Name)
}

// center は名前を (row, col) を真ん中にして置く。入らなければ近い順に、
// 上下に 1〜3 行・左右に名前の幅の半分ずらしたところを試す。どこにも入らなければ置かない。
func (g *grid) center(row, col int, name string) {
	n := Width(name)
	c := col - n/2
	for _, d := range centerTries(n) {
		if g.free(row+d[0], c+d[1], n) {
			g.put(row+d[0], c+d[1], name)
			return
		}
	}
}

// centerTries は center が試すずらし方 (行, 列) を近い順に返す。
func centerTries(n int) [][2]int {
	out := [][2]int{{0, 0}}
	for d := 1; d <= 3; d++ {
		out = append(out, [2]int{-d, 0}, [2]int{d, 0})
	}
	for _, dc := range []int{-(n/2 + 1), n/2 + 1} {
		out = append(out, [2]int{0, dc})
		for d := 1; d <= 3; d++ {
			out = append(out, [2]int{-d, dc}, [2]int{d, dc})
		}
	}
	return out
}

// insideHorizon は (row, col) が地平線の楕円の内側か。仕様: spec/sky.smt2 の 5。
func insideHorizon(row, col, rx, ry, cx, cy int) bool {
	dr, dc := row-cy, col-cx
	return dr*dr*rx*rx+dc*dc*ry*ry < rx*rx*ry*ry
}

// lineGlyph は行のずれ dr (下が正)・列のずれ dc の線を描く文字。
// 文字は縦長 (1 行 ≒ 2 列) なので、見た目の縦は 2|dr| として傾きを見る。仕様: spec/sky.smt2 の 6。
func lineGlyph(dr, dc int) rune {
	vx, vy := abs(dc), 2*abs(dr)
	switch {
	case 5*vy <= 2*vx:
		return '-'
	case 5*vx <= 2*vy:
		return '|'
	case (dr < 0) == (dc > 0):
		return '/'
	default:
		return '\\'
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// step は x の向き (1・0・-1)。
func step(x int) int {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	}
	return 0
}

// constellationLine は星座線 1 本を、両端の星の升を結んで描く。
func (g *grid) constellationLine(v view, s Segment) {
	r0, c0 := v.at(s.A)
	r1, c1 := v.at(s.B)
	g.segment(r0, c0, r1, c1, v.rx, v.ry, v.cx, v.cy)
}

// segment は (r0, c0) から (r1, c1) へ線を引く (ブレゼンハム)。地平線の内側の空いた升にだけ置く。
func (g *grid) segment(r0, c0, r1, c1, rx, ry, cx, cy int) {
	ch := lineGlyph(r1-r0, c1-c0)
	dc, dr := abs(c1-c0), -abs(r1-r0)
	sc, sr := step(c1-c0), step(r1-r0)
	e := dc + dr
	r, c := r0, c0
	for {
		if insideHorizon(r, c, rx, ry, cx, cy) && g.cells[r][c].k == empty {
			g.cells[r][c] = cell{r: ch, k: line}
		}
		if r == r1 && c == c1 {
			return
		}
		e2 := 2 * e
		if e2 >= dr {
			e += dr
			c += sc
		}
		if e2 <= dc {
			e += dc
			r += sr
		}
	}
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
