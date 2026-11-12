package sky

import (
	"testing"
	"time"
)

// 仕様 spec/sky.smt2 の性質を、実装に対して全件で確かめる。

// TestSpec_Layout: 1. 地平線の楕円が格子に収まり、半径は 1 以上。
func TestSpec_Layout(t *testing.T) {
	n := 0
	for w := 20; w <= 400; w++ {
		for h := 6; h <= 199; h++ {
			rx, ry, cx, cy := layout(w, h)
			if ry < 1 || rx < 1 || cx-rx < 0 || cx+rx > w-1 || cy-ry < 0 || cy+ry > h-1 {
				t.Fatalf("%dx%d: rx %d ry %d cx %d cy %d", w, h, rx, ry, cx, cy)
			}
			n++
		}
	}
	if n != 381*194 {
		t.Fatal(n)
	}
}

func TestLayoutAspect(t *testing.T) {
	// 幅が足りるときは縦横比 2:1、足りないときは幅いっぱい。
	// 境界: ちょうど幅が足りる (2*ry == lim) 45 列は 2:1 のまま、その 1 つ外の 44 列は幅に合わせる
	for _, c := range [][4]int{{80, 23, 22, 11}, {21, 23, 10, 5}, {45, 23, 22, 11}, {44, 23, 21, 10}} {
		if rx, ry, _, _ := layout(c[0], c[1]); rx != c[2] || ry != c[3] {
			t.Fatalf("layout(%d, %d) = %d, %d", c[0], c[1], rx, ry)
		}
	}
}

// TestSpec_SymbolMonotonic: 2. 明るいほど目立つ文字。
func TestSpec_SymbolMonotonic(t *testing.T) {
	rank := map[rune]int{'●': 3, '*': 2, '+': 1}
	n := 0
	for i := -200; i <= 300; i++ {
		for j := i; j <= 300; j++ {
			a, b := float64(i)/100, float64(j)/100
			if rank[Symbol(a)] < rank[Symbol(b)] || rank[Symbol(b)] == 0 {
				t.Fatalf("Symbol(%v)=%c, Symbol(%v)=%c", a, Symbol(a), b, Symbol(b))
			}
			n++
		}
	}
	if n != 501*502/2 {
		t.Fatal(n)
	}
}

// TestSpec_ProjectInside: 3. 高度 0〜90 の星は楕円の中 (|行 - cy| <= ry・|列 - cx| <= rx)。
func TestSpec_ProjectInside(t *testing.T) {
	n := 0
	for _, wh := range [][2]int{{20, 6}, {20, 7}, {21, 11}, {44, 23}, {60, 19}, {80, 23}, {400, 199}} {
		rx, ry, cx, cy := layout(wh[0], wh[1])
		for alt := 0.0; alt <= 90; alt += 0.5 {
			for az := 0.0; az < 360; az += 1 {
				row, col := project(alt, az, rx, ry, cx, cy)
				if row < cy-ry || row > cy+ry || col < cx-rx || col > cx+rx {
					t.Fatalf("%v: 高度 %v 方位 %v → (%d, %d)", wh, alt, az, row, col)
				}
				n++
			}
		}
	}
	if n != 7*181*360 {
		t.Fatal(n)
	}
	// 天頂は中心、高度 0 の北は上の縁、東は左の縁
	if r, c := project(90, 0, 22, 11, 39, 11); r != 11 || c != 39 {
		t.Fatal(r, c)
	}
	if r, c := project(0, 0, 22, 11, 39, 11); r != 0 || c != 39 {
		t.Fatal(r, c)
	}
	if r, c := project(0, 90, 22, 11, 39, 11); r != 11 || c != 17 {
		t.Fatal(r, c)
	}
}

// TestSpec_VisibleAboveHorizon: 描くのは地平線より上の星だけ、明るい順。
func TestSpec_VisibleAboveHorizon(t *testing.T) {
	c := tokyo(t)
	n := 0
	for d := 0; d < 365; d += 5 {
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(d)*24*time.Hour + time.Duration(d%24)*time.Hour)
		seen := Visible(Catalog, c, at)
		for i, s := range seen {
			if s.Alt <= 0 || (i > 0 && seen[i-1].Star.Mag > s.Star.Mag) {
				t.Fatalf("%v: %+v", at, s)
			}
		}
		n++
	}
	if n != 73 {
		t.Fatal(n)
	}
}

// inGridAndEllipseBox は (row, col) が幅 w・高さ h の格子の中で、楕円を囲む四角の中か。
func inGridAndEllipseBox(row, col, w, h int) bool {
	rx, ry, cx, cy := layout(w, h)
	return row >= 0 && row < h && col >= 0 && col < w && row >= cy-ry && row <= cy+ry && col >= cx-rx && col <= cx+rx
}

// checkSegmentsInside は幅 w・高さ h の格子で、端点 (上下左右に格子 1 つ分外まで、高さ・幅の約 4 分の 1 刻み) の全部の組に線を引き、
// 線の升がすべて地平線の楕円の内側にあることと、内側を通る線が少なくとも 1 升は描かれることを確かめる。
func checkSegmentsInside(t *testing.T, w, h int) {
	t.Helper()
	rx, ry, cx, cy := layout(w, h)
	var pts [][2]int
	for r := -h; r < 2*h; r += 1 + h/4 {
		for c := -w; c < 2*w; c += 1 + w/4 {
			pts = append(pts, [2]int{r, c})
		}
	}
	drawnAny := 0
	for _, p := range pts {
		for _, q := range pts {
			g := newGrid(w, h)
			g.segment(p[0], p[1], q[0], q[1], rx, ry, cx, cy)
			for row := range g.cells {
				for col, c := range g.cells[row] {
					if c.k == line && !insideHorizon(row, col, rx, ry, cx, cy) {
						t.Fatalf("%dx%d: %v→%v の線が (%d, %d) で地平線の外", w, h, p, q, row, col)
					}
					if c.k == line {
						drawnAny++
					}
				}
			}
		}
	}
	if drawnAny == 0 {
		t.Fatalf("%dx%d: 線が 1 升も描かれない", w, h)
	}
}

// TestSpec_LineInsideHorizon: 5. 楕円の内側の点は、どの大きさの格子にも収まる (線は格子からはみ出さない)。
func TestSpec_LineInsideHorizon(t *testing.T) {
	n, want := 0, 0
	for _, wh := range [][2]int{{20, 6}, {20, 7}, {21, 11}, {44, 23}, {60, 19}, {80, 23}, {131, 59}} {
		w, h := wh[0], wh[1]
		want += 9 * w * h
		rx, ry, cx, cy := layout(w, h)
		for row := -h; row < 2*h; row++ {
			for col := -w; col < 2*w; col++ {
				if insideHorizon(row, col, rx, ry, cx, cy) && !inGridAndEllipseBox(row, col, w, h) {
					t.Fatalf("%v: (%d, %d) は内側なのに格子・楕円の外", wh, row, col)
				}
				n++
			}
		}
		// 実装: 格子の外まで含む端点の組で g.segment を呼び、置かれた線の升はどれも楕円の内側
		checkSegmentsInside(t, w, h)
		// 中心は内側、縁 (N・E の升) は内側でない
		if !insideHorizon(cy, cx, rx, ry, cx, cy) || insideHorizon(cy-ry, cx, rx, ry, cx, cy) || insideHorizon(cy, cx+rx, rx, ry, cx, cy) {
			t.Fatalf("%v: 中心か縁", wh)
		}
	}
	if n != want {
		t.Fatal(n)
	}
}

// TestSpec_LineGlyph: 6. 線の文字は見た目の傾きと合う (行のずれ・列のずれ -60〜60 の全部)。
func TestSpec_LineGlyph(t *testing.T) {
	n := 0
	for dr := -60; dr <= 60; dr++ {
		for dc := -60; dc <= 60; dc++ {
			vx, vy := abs(dc), 2*abs(dr)
			ok := false
			switch lineGlyph(dr, dc) {
			case '-':
				ok = 5*vy <= 2*vx
			case '|':
				ok = 5*vx <= 2*vy && dr != 0
			case '/':
				ok = dr*dc < 0
			case '\\':
				ok = dr*dc > 0
			}
			if !ok {
				t.Fatalf("lineGlyph(%d, %d) = %c", dr, dc, lineGlyph(dr, dc))
			}
			n++
		}
	}
	if n != 121*121 {
		t.Fatal(n)
	}
	// 代表の向き (行は下が正) と、'-' と '/' の境目 (5・2|dr| = 2|dc|) の両側
	for _, c := range []struct {
		dr, dc int
		want   rune
	}{{0, 5, '-'}, {-5, 0, '|'}, {-2, 4, '/'}, {2, 4, '\\'}, {-2, -4, '\\'}, {2, -4, '/'}, {0, 0, '-'}, {1, 5, '-'}, {-1, 4, '/'}, {5, 1, '|'}, {4, 1, '|'}} {
		if got := lineGlyph(c.dr, c.dc); got != c.want {
			t.Fatalf("lineGlyph(%d, %d) = %c, want %c", c.dr, c.dc, got, c.want)
		}
	}
}
