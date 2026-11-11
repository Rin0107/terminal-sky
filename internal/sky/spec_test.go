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
		for h := 7; h <= 199; h++ {
			rx, ry, cx, cy := layout(w, h)
			if ry < 1 || rx < 1 || cx-rx < 0 || cx+rx > w-1 || cy-ry < 0 || cy+ry > h-1 {
				t.Fatalf("%dx%d: rx %d ry %d cx %d cy %d", w, h, rx, ry, cx, cy)
			}
			n++
		}
	}
	if n != 381*193 {
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
	for _, wh := range [][2]int{{20, 7}, {21, 11}, {44, 23}, {60, 19}, {80, 23}, {400, 199}} {
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
	if n != 6*181*360 {
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
