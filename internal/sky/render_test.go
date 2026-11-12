package sky

import (
	"math"
	"strings"
	"testing"
	"time"
)

// AC-1-4: 等級が明るい星ほど目立つ文字で描かれる (● > * > +)。
func TestAC_1_4_SymbolByMagnitude(t *testing.T) {
	rank := map[rune]int{'●': 3, '*': 2, '+': 1}
	// 境界値: 0.5 と 1.5 の端とその 1 つ外
	cases := []struct {
		mag  float64
		want rune
	}{{-1.46, '●'}, {0.49, '●'}, {0.5, '*'}, {1.49, '*'}, {1.5, '+'}, {1.99, '+'}}
	for _, c := range cases {
		if got := Symbol(c.mag); got != c.want {
			t.Fatalf("AC-1-4: Symbol(%v) = %c, want %c", c.mag, got, c.want)
		}
	}
	// 全件: 明るい (等級が小さい) ほど目立ち方が下がらない
	prev, n := 99, 0
	for m := -200; m < 200; m++ {
		r := rank[Symbol(float64(m)/100)]
		if r == 0 || r > prev {
			t.Fatalf("AC-1-4: 等級 %v で順が崩れた", float64(m)/100)
		}
		prev = r
		n++
	}
	if n != 400 {
		t.Fatal(n)
	}
}

// AC-1-4: 図の中でも、等級どおりの文字で描かれる。
// 架空の星 3 つ: A (-1 等) を天頂、B (1 等) を東の高度 45 度、C (1.8 等) を西の高度 45 度に置くと、
// 真ん中の行に 左から * (B) ・ ● (A) ・ + (C) と並ぶ。
func TestAC_1_4_ChartUsesSymbols(t *testing.T) {
	seen := []Seen{
		{Star: Star{Name: "A", Mag: -1}, Alt: 90, Az: 0},
		{Star: Star{Name: "B", Mag: 1}, Alt: 45, Az: 90},
		{Star: Star{Name: "C", Mag: 1.8}, Alt: 45, Az: 270},
	}
	row := Chart(seen, 41, 21)[10]
	b, a, c := strings.IndexRune(row, '*'), strings.IndexRune(row, '●'), strings.IndexRune(row, '+')
	if b < 0 || a < 0 || c < 0 || b >= a || a >= c {
		t.Fatalf("AC-1-4: * %d ● %d + %d: %q", b, a, c, row)
	}
}

// AC-1-4: 本物の空でも、シリウス (-1.46 等) は ●、アダーラ (1.50 等) は + で描かれる。
func TestAC_1_4_RealSky(t *testing.T) {
	seen := Visible(Catalog, tokyo(t), jan15)
	rx, ry, cx, cy := layout(80, 23)
	lines := Chart(seen, 80, 23)
	checked := 0
	defer func() {
		if checked != 2 {
			t.Fatalf("AC-1-4: 確かめた星 %d (2 のはず)", checked)
		}
	}()
	for _, s := range seen {
		want := map[string]rune{"シリウス": '●', "アダーラ": '+'}[s.Star.Name]
		if want == 0 {
			continue
		}
		checked++
		row, col := project(s.Alt, s.Az, rx, ry, cx, cy)
		// 行の中の位置は、全角の文字があると列とずれるので、列までの幅を数えて探す
		got, w := rune(0), 0
		for _, r := range lines[row] {
			if w == col {
				got = r
				break
			}
			w += RuneWidth(r)
		}
		if got != want {
			t.Fatalf("AC-1-4: %s は %c のはずが %c\n%s", s.Star.Name, want, got, lines[row])
		}
	}
}

func TestChartSizeAndOrientation(t *testing.T) {
	seen := []Seen{
		{Star: Star{Name: "北", Mag: 1}, Alt: 10, Az: 0},
		{Star: Star{Name: "南", Mag: 1}, Alt: 10, Az: 180},
	}
	lines := Chart(seen, 80, 23)
	if len(lines) != 23 {
		t.Fatal(len(lines))
	}
	if !strings.Contains(lines[0], "N") || !strings.Contains(lines[22], "S") {
		t.Fatalf("北が上・南が下でない:\n%s", strings.Join(lines, "\n"))
	}
	if !strings.Contains(lines[11], "E") || strings.Index(lines[11], "E") > strings.Index(lines[11], "W") {
		t.Fatalf("東が左・西が右でない: %q", lines[11])
	}
	north, south := -1, -1
	for i, l := range lines {
		if strings.Contains(l, "北") {
			north = i
		}
		if strings.Contains(l, "南") {
			south = i
		}
	}
	if north < 0 || south < 0 || north >= 11 || south <= 11 {
		t.Fatalf("北 %d 南 %d\n%s", north, south, strings.Join(lines, "\n"))
	}
}

func TestChartBrighterWinsSameCell(t *testing.T) {
	seen := []Seen{
		{Star: Star{Name: "明", Mag: 0}, Alt: 90},
		{Star: Star{Name: "暗", Mag: 1.9}, Alt: 90},
	}
	out := strings.Join(Chart(seen, 41, 21), "\n")
	if !strings.Contains(out, "●") || strings.Contains(out, "+") || strings.Contains(out, "暗") || !strings.Contains(out, "明") {
		t.Fatal(out)
	}
}

// 名前は 右 → 左 → 右上 → 右下 → 左上 → 左下 の順に試し、どこにも入らなければ出さない。
func TestLabelPlacement(t *testing.T) {
	// 7 行 × 12 列の格子の (3, 5) に星があり、名前は "AB" (幅 2)
	setup := func(block ...[2]int) *grid {
		g := newGrid(12, 7)
		g.cells[3][5] = cell{r: '*', k: solid}
		for _, b := range block {
			g.cells[b[0]][b[1]] = cell{r: '#', k: solid}
		}
		return g
	}
	cases := []struct {
		name  string
		block [][2]int
		want  [2]int // 置かれる場所 (行, 列)。-1 は置かれない
	}{
		{"右", nil, [2]int{3, 7}},
		{"左", [][2]int{{3, 8}}, [2]int{3, 2}},
		{"右上", [][2]int{{3, 8}, {3, 1}}, [2]int{2, 6}},
		{"右下", [][2]int{{3, 8}, {3, 1}, {2, 8}}, [2]int{4, 6}},
		{"左上", [][2]int{{3, 8}, {3, 1}, {2, 8}, {4, 7}}, [2]int{2, 2}},
		{"左下", [][2]int{{3, 8}, {3, 1}, {2, 8}, {4, 7}, {2, 1}}, [2]int{4, 2}},
		{"なし", [][2]int{{3, 8}, {3, 1}, {2, 8}, {4, 7}, {2, 1}, {4, 1}}, [2]int{-1, -1}},
	}
	for _, c := range cases {
		g := setup(c.block...)
		g.label(3, 5, "AB")
		got := [2]int{-1, -1}
		for r := range g.cells {
			for col := range g.cells[r] {
				if g.cells[r][col].r == 'A' {
					got = [2]int{r, col}
				}
			}
		}
		if got != c.want {
			t.Fatalf("%s: 置かれた場所 %v, want %v\n%s", c.name, got, c.want, strings.Join(g.lines(), "\n"))
		}
	}
}

func TestGridFree(t *testing.T) {
	g := newGrid(10, 3)
	g.cells[1][4] = cell{r: '.', k: horizon}
	if !g.free(1, 3, 3) {
		t.Fatal("地平線の点の上には置ける")
	}
	g.put(1, 0, "東") // (1,0) と (1,1) を使う
	// 境界: 端とその 1 つ外
	for _, c := range []struct {
		row, col, n int
		want        bool
	}{
		{0, 0, 10, true}, {0, 0, 11, false}, {0, -1, 1, false}, {2, 9, 1, true}, {2, 10, 1, false},
		{-1, 0, 1, false}, {3, 0, 1, false}, {1, 3, 2, true}, {1, 2, 2, false}, {1, 1, 1, false},
	} {
		if got := g.free(c.row, c.col, c.n); got != c.want {
			t.Fatalf("free(%d,%d,%d) = %v", c.row, c.col, c.n, got)
		}
	}
	if l := g.lines(); l[1] != "東  ." || l[0] != "" {
		t.Fatalf("%q", l)
	}
}

func TestChartLongNameNotPlaced(t *testing.T) {
	l := Chart([]Seen{{Star: Star{Name: strings.Repeat("長", 30), Mag: 1}, Alt: 45}}, 21, 11)
	for _, row := range l {
		if strings.Contains(row, "長") || Width(row) > 21 {
			t.Fatalf("入らない名前が出た: %q", row)
		}
	}
}

func TestWidthAndTruncate(t *testing.T) {
	cases := []struct {
		s string
		w int
	}{{"", 0}, {"abc", 3}, {"東京", 4}, {"ベテルギウス", 12}, {"●*+", 3}, {"ｱ", 1}, {"Ａ", 2}, {"한", 2}, {"ᄀ", 2}, {"豈", 2}, {"︰", 2}, {"￠", 2}}
	for _, c := range cases {
		if got := Width(c.s); got != c.w {
			t.Fatalf("Width(%q) = %d, want %d", c.s, got, c.w)
		}
	}
	// 境界: 各範囲の端の 1 つ外は幅 1
	for _, r := range []rune{0x10FF, 0x1160, 0x2E7F, 0xA4D0, 0xABFF, 0xD7A4, 0xF8FF, 0xFB00, 0xFE2F, 0xFE50, 0xFEFF, 0xFF61, 0xFFDF, 0xFFE7} {
		if RuneWidth(r) != 1 {
			t.Fatalf("%U は幅 1", r)
		}
	}
	for _, r := range []rune{0x1100, 0x115F, 0x2E80, 0xA4CF, 0xAC00, 0xD7A3, 0xF900, 0xFAFF, 0xFE30, 0xFE4F, 0xFF00, 0xFF60, 0xFFE0, 0xFFE6} {
		if RuneWidth(r) != 2 {
			t.Fatalf("%U は幅 2", r)
		}
	}
	if got := Truncate("東京abc", 3); got != "東" {
		t.Fatal(got)
	}
	if got := Truncate("東京abc", 4); got != "東京" {
		t.Fatal(got)
	}
	if got := Truncate("東京abc", 7); got != "東京abc" {
		t.Fatal(got)
	}
	if got := Truncate("東京abc", 6); got != "東京ab" {
		t.Fatal(got)
	}
}

// countLine は格子の中の、文字 ch の星座線の升の数。
func countLine(g *grid, ch rune) int {
	n := 0
	for _, row := range g.cells {
		for _, c := range row {
			if c.r == ch && c.k == line {
				n++
			}
		}
	}
	return n
}

// 線はブレゼンハムでつながる。41 × 21 の格子: 中心 (10, 20)・半径 rx 20・ry 10。
func TestSegment(t *testing.T) {
	rx, ry, cx, cy := layout(41, 21)
	if rx != 20 || ry != 10 || cx != 20 || cy != 10 {
		t.Fatal(rx, ry, cx, cy)
	}
	count := countLine
	// 水平: 両端を含めて 11 升
	g := newGrid(41, 21)
	g.segment(10, 15, 10, 25, rx, ry, cx, cy)
	if count(g, '-') != 11 {
		t.Fatalf("水平 %d\n%s", count(g, '-'), strings.Join(g.lines(), "\n"))
	}
	// 垂直 (上向き)・右上がり・左下がり: どの升も前の升と隣り合う (途切れない)
	for _, c := range []struct {
		r0, c0, r1, c1 int
		ch             rune
		n              int
	}{{14, 20, 6, 20, '|', 9}, {12, 14, 8, 22, '/', 9}, {8, 22, 12, 14, '/', 9}, {8, 14, 12, 22, '\\', 9}} {
		g := newGrid(41, 21)
		g.segment(c.r0, c.c0, c.r1, c.c1, rx, ry, cx, cy)
		if got := count(g, c.ch); got != c.n || g.cells[c.r0][c.c0].r != c.ch || g.cells[c.r1][c.c1].r != c.ch {
			t.Fatalf("%v: %d 升\n%s", c, got, strings.Join(g.lines(), "\n"))
		}
	}
}

// 線は地平線の内側の空いた升にだけ置かれる。
func TestSegmentClipAndKeep(t *testing.T) {
	rx, ry, cx, cy := layout(41, 21)
	count := countLine
	// 地平線の外 (楕円の外・格子の外) へ伸びる線は、内側の部分だけ描く
	g := newGrid(41, 21)
	g.segment(10, 20, 10, 60, rx, ry, cx, cy)
	if count(g, '-') != 20 || g.cells[10][39].k != line || g.cells[10][40].k != empty {
		t.Fatalf("外へ伸びる線 %d\n%s", count(g, '-'), strings.Join(g.lines(), "\n"))
	}
	// 星・地平線・名前の升は上書きしない
	g = newGrid(41, 21)
	g.cells[10][17] = cell{r: '*', k: solid}
	g.cells[10][18] = cell{r: '.', k: horizon}
	g.put(10, 19, "東")
	g.segment(10, 15, 10, 25, rx, ry, cx, cy)
	if l := g.lines()[10]; l != "               --*.東-----" {
		t.Fatalf("%q", l)
	}
}

// 星座の名前は、見えている星の真ん中に置き、入らなければ近くにずらす。どこにも入らなければ出さない。
func TestCenterPlacement(t *testing.T) {
	where := func(g *grid) [2]int {
		for r := range g.cells {
			for c := range g.cells[r] {
				if g.cells[r][c].r == 'A' {
					return [2]int{r, c}
				}
			}
		}
		return [2]int{-1, -1}
	}
	// 9 行 × 20 列。(4, 10) を真ん中に "ABCD" (幅 4) を置くと、列 8 から
	g := newGrid(20, 9)
	g.center(4, 10, "ABCD")
	if got := where(g); got != [2]int{4, 8} {
		t.Fatalf("真ん中 %v", got)
	}
	// 真ん中の行がふさがっていれば 1 行上
	g = newGrid(20, 9)
	g.cells[4][9] = cell{r: '*', k: solid}
	g.center(4, 10, "ABCD")
	if got := where(g); got != [2]int{3, 8} {
		t.Fatalf("1 行上 %v", got)
	}
	// 列 10 が上下 3 行ともふさがっていれば、左へ (幅の半分 + 1) ずらす
	g = newGrid(20, 9)
	for r := 1; r <= 7; r++ {
		g.cells[r][10] = cell{r: '*', k: solid}
	}
	g.center(4, 10, "ABCD")
	if got := where(g); got != [2]int{4, 5} {
		t.Fatalf("左 %v\n%s", got, strings.Join(g.lines(), "\n"))
	}
	// 入るところがなければ置かない
	g = newGrid(6, 3)
	g.center(1, 3, "ABCDEFGH")
	if got := where(g); got != [2]int{-1, -1} {
		t.Fatalf("置かれた %v", got)
	}
	// 試す場所は近い順に 21 か所で、重ならない
	tries := centerTries(4)
	seen := map[[2]int]bool{}
	for _, d := range tries {
		if seen[d] {
			t.Fatalf("同じ場所を 2 度試す: %v", d)
		}
		seen[d] = true
	}
	if len(tries) != 21 || tries[0] != [2]int{0, 0} {
		t.Fatal(tries)
	}
}

// 星座: 線・線の星の点・星座の名前を描く。NoLines なら線だけ消える。
func TestDrawFigures(t *testing.T) {
	// 天頂の明るい星 A と、東の高度 60 度の暗い星 B を結ぶ「てすと座」
	a := Seen{Star: Star{Name: "A", Mag: 0}, Alt: 90}
	b := Seen{Star: Star{Name: "B", Mag: 3}, Alt: 60, Az: 90}
	fig := Figure{Name: "てすと座", Segments: []Segment{{A: a, B: b}}, Stars: []Seen{a, b}}
	lines := Draw(Scene{Stars: []Seen{a}, Figures: []Figure{fig}}, 81, 41)
	out := strings.Join(lines, "\n")
	// 真ん中の行 (20) に B の + ・線 ・ A の ● が並ぶ。暗い星 B に名前は付けない
	row := lines[20]
	if !strings.Contains(row, "+--") || !strings.Contains(row, "--● A") || strings.Contains(out, "B") || !strings.Contains(out, "A") || !strings.Contains(out, "てすと座") {
		t.Fatalf("\n%s", out)
	}
	off := strings.Join(Draw(Scene{Stars: []Seen{a}, Figures: []Figure{fig}, NoLines: true}, 81, 41), "\n")
	if strings.ContainsAny(off, `-|/\`) || !strings.Contains(off, "+") || !strings.Contains(off, "てすと座") {
		t.Fatalf("NoLines:\n%s", off)
	}
}

// 2 等星の名前は星座の名前に場所を譲り、1 等星の名前は譲らない。
func TestDrawLabelPriority(t *testing.T) {
	// 41 × 21 の格子 (中心 (10, 20))。天頂の星の名前は右の (10, 22) に入る。
	// 星座「ざ」の点を (8, 23) と (12, 23) に置くと、名前の真ん中も (10, 23) で、同じ (10, 22) を取り合う。
	at := func(dr, dc float64) Seen { // 中心からのずれ (行, 列) の位置の高度・方位
		r := math.Hypot(dr/10, dc/20)
		return Seen{Star: Star{Name: "点", Mag: 3}, Alt: 90 - 90*r, Az: math.Atan2(-dc/20, -dr/10) / deg}
	}
	fig := Figure{Name: "ざ", Stars: []Seen{at(-2, 3), at(2, 3)}}
	for _, c := range []struct {
		mag  float64
		want int // 星の名前が置かれる列 (右に入れば 22、左へ回れば 17)
	}{{1.49, 22}, {1.5, 17}} {
		s := Seen{Star: Star{Name: "QQ", Mag: c.mag}, Alt: 90}
		lines := Draw(Scene{Stars: []Seen{s}, Figures: []Figure{fig}}, 41, 21)
		row := lines[10]
		if got := strings.Index(row, "QQ"); got != c.want || !strings.Contains(strings.Join(lines, "\n"), "ざ") {
			t.Fatalf("等級 %v: QQ の列 %d, want %d\n%s", c.mag, got, c.want, strings.Join(lines, "\n"))
		}
	}
}

// 本物の星座を、いろいろな大きさ・時刻で描いても、幅と行数を守る (全角の文字の右半分を壊さない)。
func TestDrawFiguresAllSizes(t *testing.T) {
	c := City{Key: "x", LatDeg: 35, LonDeg: 139, Zone: jst}
	n := 0
	for hr := 0; hr < 24; hr += 3 {
		at := jan15.Add(time.Duration(hr) * time.Hour)
		sc := Scene{Stars: Visible(Catalog, c, at), Figures: Figures(Constellations, Stars(), c, at)}
		for w := 20; w <= 130; w += 7 {
			for h := 7; h <= 60; h += 5 {
				l := Draw(sc, w, h)
				if len(l) != h {
					t.Fatalf("%dx%d: %d 行", w, h, len(l))
				}
				for _, row := range l {
					if Width(row) > w || len([]rune(row)) > w {
						t.Fatalf("%v %dx%d: 幅を超えた %q", at, w, h, row)
					}
				}
				n++
			}
		}
	}
	if n != 8*16*11 {
		t.Fatal(n)
	}
}

func TestChartAllSizes(t *testing.T) {
	seen := Visible(Catalog, City{Key: "x", LatDeg: 35, LonDeg: 139, Zone: jst}, jan15)
	n := 0
	for w := 20; w <= 130; w += 3 {
		for h := 7; h <= 60; h += 2 {
			l := Chart(seen, w, h)
			if len(l) != h {
				t.Fatalf("%dx%d: %d 行", w, h, len(l))
			}
			for _, row := range l {
				if Width(row) > w || len([]rune(row)) > w {
					t.Fatalf("%dx%d: 幅を超えた %q", w, h, row)
				}
			}
			n++
		}
	}
	if n != 37*27 {
		t.Fatal(n)
	}
}
