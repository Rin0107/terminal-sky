package sky

import (
	"math"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestParseFacing(t *testing.T) {
	for _, c := range []struct {
		in    string
		want  Facing
		ok    bool
		label string
	}{
		{"north", North, true, "北"}, {"East", East, true, "東"}, {" SOUTH ", South, true, "南"}, {"west", West, true, "西"},
		{"", AllSky, false, ""}, {"up", AllSky, false, ""}, {"n", AllSky, false, ""}, {"northeast", AllSky, false, ""},
	} {
		got, ok := ParseFacing(c.in)
		if got != c.want || ok != c.ok || got.Label() != c.label {
			t.Fatalf("ParseFacing(%q) = %v %v %q", c.in, got, ok, got.Label())
		}
	}
	if !slices.Equal(FacingKeys(), []string{"north", "east", "south", "west"}) {
		t.Fatal(FacingKeys())
	}
	// 呼んだ側が変えても一覧は変わらない
	FacingKeys()[0] = "x"
	if FacingKeys()[0] != "north" {
		t.Fatal("一覧が書き換わった")
	}
	for f, az := range map[Facing]float64{North: 0, East: 90, South: 180, West: 270} {
		if f.az() != az {
			t.Fatalf("%v: %v", f, f.az())
		}
	}
}

// 外から来る --facing の文字列: どんな文字列でも止まらず、読めたら North〜West で、FacingKeys の名前に往復できる。
// 読めなければ AllSky (呼ぶ側は誤りの文を返す)。
func FuzzParseFacing(f *testing.F) {
	for _, s := range []string{"north", " WEST ", "", "up", "n", "\xff", "southsouth"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got, ok := ParseFacing(s)
		if !ok {
			if got != AllSky {
				t.Fatalf("%q: %v", s, got)
			}
			return
		}
		if got < North || got > West || FacingKeys()[got-North] != strings.ToLower(strings.TrimSpace(s)) {
			t.Fatalf("%q: %v", s, got)
		}
		if back, ok := ParseFacing(FacingKeys()[got-North]); !ok || back != got || got.Label() == "" {
			t.Fatalf("%q: 往復 %v", s, back)
		}
	})
}

// TestSpec_RelAz: 12. 向きから見た角は -180 以上 180 未満で、方位と 360 の倍数だけ違う。
// 向きの真後ろをまたぐ 2 点 (差が 180 より大きい) の線は描かない。
func TestSpec_RelAz(t *testing.T) {
	n := 0
	for c := 0.0; c < 360; c += 90 {
		for az := -360.0; az <= 720; az += 0.25 {
			r := relAz(az, c)
			k := (az - c - r) / 360
			if r < -180 || r >= 180 || math.Abs(k-math.Round(k)) > 1e-9 {
				t.Fatalf("relAz(%v, %v) = %v", az, c, r)
			}
			n++
		}
	}
	if n != 4*4321 {
		t.Fatal(n)
	}
	// 境界: 真後ろは -180、左右 90 度の縁は ±90
	for _, c := range [][3]float64{{0, 180, -180}, {180, 0, -180}, {180, 90, -90}, {180, 270, 90}, {0, 359.75, -0.25}} {
		if got := relAz(c[1], c[0]); got != c[2] {
			t.Fatalf("relAz(%v, %v) = %v, want %v", c[1], c[0], got, c[2])
		}
	}
	v := newFacingView(North, 81, 23)
	at := func(az float64) Seen { return Seen{Alt: 30, Az: az} }
	// 北を向いて: 170 度と 190 度・90 度と 269 度は南 (背中) を通る。10 度と 350 度・80 度と 300 度は北 (前) を通る
	if !v.crosses(at(170), at(190)) || !v.crosses(at(90), at(269)) || v.crosses(at(10), at(350)) || v.crosses(at(80), at(300)) {
		t.Fatal("背中側をまたぐ線の判定")
	}
}

// TestSpec_FacingInside: 11. 方角を向いた図に入るもの (高度 0 より上・向きから左右 90 度以内) は、
// 地平線の行より上 (0〜bottom 行) で、左右の縁 (cx ± hx) の中に置かれ、格子からはみ出さない。
func TestSpec_FacingInside(t *testing.T) {
	n := 0
	for _, wh := range [][2]int{{20, 6}, {21, 7}, {40, 10}, {80, 22}, {81, 23}, {400, 199}} {
		w, h := wh[0], wh[1]
		for f := North; f <= West; f++ {
			v := newFacingView(f, w, h)
			if v.cx-v.hx < 0 || v.cx+v.hx > w-1 || v.bottom != h-2 {
				t.Fatalf("%v: %+v", wh, v)
			}
			n += checkFacingInside(t, f, v)
		}
	}
	if n != 6*4*360*361 {
		t.Fatal(n)
	}
}

// checkFacingInside は向き f の図 v で、高度 0.25〜90 (0.25 度刻み)・向きから -90〜90 度 (0.5 度刻み) の全部が
// 図に入り、地平線の行より上・左右の縁の中に置かれることを確かめ、確かめた数を返す。
func checkFacingInside(t *testing.T, f Facing, v facingView) int {
	t.Helper()
	n := 0
	for alt := 0.25; alt <= 90; alt += 0.25 {
		for rel := -90.0; rel <= 90; rel += 0.5 {
			az := f.az() + rel
			row, col := v.pos(alt, az)
			if !v.shows(alt, az) || !f.Shows(alt, az) || row < 0 || row > v.bottom || abs(col-v.cx) > v.hx {
				t.Fatalf("%v %+v: 高度 %v 向きから %v → (%d, %d)", f, v, alt, rel, row, col)
			}
			n++
		}
	}
	return n
}

// 方角を向いた図の境目と、代表の位置。
func TestFacingEdges(t *testing.T) {
	// 図に入らないもの: 地平線 (高度 0) とその下、向きから 90 度より外
	v := newFacingView(South, 81, 23)
	for _, c := range [][2]float64{{0, 180}, {-5, 180}, {45, 89.75}, {45, 270.25}, {45, 0}} {
		if v.shows(c[0], c[1]) || South.Shows(c[0], c[1]) {
			t.Fatalf("高度 %v 方位 %v が図に入った", c[0], c[1])
		}
	}
	// 全天の図は地平線より上の全部
	if !AllSky.Shows(1, 0) || AllSky.Shows(0, 0) {
		t.Fatal("全天")
	}
	// 天頂は真ん中の上の縁、地平線の向きは真ん中の高度 0 の行、左の縁は向きの 90 度左
	for _, c := range []struct {
		alt, az  float64
		row, col int
	}{{90, 0, 0, 40}, {0.001, 180, 21, 40}, {0.001, 90, 21, 0}, {0.001, 270, 21, 80}, {60, 90, 7, 20}} {
		if r, col := v.pos(c.alt, c.az); r != c.row || col != c.col {
			t.Fatalf("pos(%v, %v) = (%d, %d), want (%d, %d)", c.alt, c.az, r, col, c.row, c.col)
		}
	}
}

// 方角を向いた図の線は、地平線の行より上・左右 90 度の縁の中にだけ置く (背中側・地平線の下へ伸びる線は縁で切る)。
func TestFacingLinesInside(t *testing.T) {
	for _, wh := range [][2]int{{20, 6}, {41, 21}, {80, 22}} {
		w, h := wh[0], wh[1]
		v := newFacingView(South, w, h)
		drawn := 0
		for _, a := range []Seen{{Alt: 80, Az: 180}, {Alt: 10, Az: 100}, {Alt: 45, Az: 0}, {Alt: -20, Az: 200}, {Alt: 30, Az: 300}} {
			for _, b := range []Seen{{Alt: 5, Az: 260}, {Alt: 70, Az: 20}, {Alt: -40, Az: 150}, {Alt: 89, Az: 90}} {
				g := newGrid(w, h)
				g.constellationLine(v, Segment{A: a, B: b})
				for row := range g.cells {
					for col, c := range g.cells[row] {
						if c.k != line {
							continue
						}
						drawn++
						if !v.inside(row, col) || row > v.bottom || abs(col-v.cx) > v.hx {
							t.Fatalf("%v: %v→%v の線が (%d, %d) で外", wh, a, b, row, col)
						}
					}
				}
			}
		}
		if drawn == 0 {
			t.Fatalf("%v: 線が 1 升も描かれない", wh)
		}
		// 天頂の行は真ん中の 1 升だけ、地平線の上の行は左右の縁まで
		if v.halfWidth(0) != 0 || v.halfWidth(v.bottom) != v.hx {
			t.Fatal(v.halfWidth(0), v.halfWidth(v.bottom))
		}
	}
	// 背中側をまたぐ線は描かない
	g := newGrid(41, 21)
	g.constellationLine(newFacingView(North, 41, 21), Segment{A: Seen{Alt: 30, Az: 170}, B: Seen{Alt: 30, Az: 190}})
	if countLine(g, '-') != 0 {
		t.Fatal(strings.Join(g.lines(), "\n"))
	}
}

// 方角を向いた図: 前の星・月・惑星と星座だけを描き、背中側のものは描かない。地平線の行に 45 度ごとの方角。
func TestDrawFacing(t *testing.T) {
	front := Seen{Star: Star{Name: "マエ", Mag: 0}, Alt: 45, Az: 180}
	back := Seen{Star: Star{Name: "ウシロ", Mag: 0}, Alt: 45, Az: 0}
	fb := Seen{Star: Star{Name: "b", Mag: 3}, Alt: 20, Az: 200}
	figs := []Figure{
		{Name: "まえ座", Segments: []Segment{{A: front, B: fb}}, Stars: []Seen{front, fb}},
		{Name: "うしろ座", Segments: []Segment{{A: back, B: Seen{Alt: 50, Az: 10}}}, Stars: []Seen{back}},
	}
	sc := Scene{
		Stars: []Seen{front, back}, Figures: figs, Facing: South,
		Moon:    MoonSeen{Alt: 30, Az: 0, Age: 15},
		Planets: []PlanetSeen{{Planet: Planet{Name: "木星", Symbol: '木'}, Alt: 30, Az: 150}, {Planet: Planet{Name: "土星", Symbol: '土'}, Alt: 30, Az: 10}},
	}
	lines := Draw(sc, 81, 23)
	out := strings.Join(lines, "\n")
	for _, want := range []string{"マエ", "まえ座", "木"} {
		if !strings.Contains(out, want) {
			t.Fatalf("%s がない\n%s", want, out)
		}
	}
	for _, not := range []string{"ウシロ", "うしろ座", "土", string(MoonSymbol(15))} {
		if strings.Contains(out, not) {
			t.Fatalf("背中側の %s が描かれた\n%s", not, out)
		}
	}
	if len(lines) != 23 || lines[22] != "E"+strings.Repeat(".", 19)+"SE"+strings.Repeat(".", 18)+"S"+strings.Repeat(".", 19)+"SW"+strings.Repeat(".", 18)+"W" {
		t.Fatalf("地平線 %q", lines[22])
	}
	// 前の星 (高度 45 度・真ん中) は真ん中の列、高さの半分の行 (21 行の半分 10.5 を四捨五入して 11)
	if r := []rune(lines[11]); len(r) <= 40 || r[40] != '●' {
		t.Fatalf("%q", lines[11])
	}
	// 北を向けば左が西・右が東
	if l := Draw(Scene{Facing: North}, 21, 7)[6]; !strings.HasPrefix(l, "W") || !strings.HasSuffix(l, "E") || !strings.Contains(l, "N") {
		t.Fatalf("%q", l)
	}
}

// 本物の空を、4 つの向き・いろいろな大きさ・時刻・都市で描いても、幅と行数を守る。
func TestDrawFacingAllSizes(t *testing.T) {
	n := 0
	for _, key := range []string{"tokyo", "naha", "sydney", "london"} {
		c, _ := FindCity(key)
		for hr := 0; hr < 24; hr += 6 {
			at := jan15.Add(time.Duration(hr) * time.Hour)
			sc := Scene{Stars: Visible(Catalog, c, at), Figures: Figures(Constellations, Stars(), c, at),
				Moon: MoonAt(c, at), Planets: VisiblePlanets(Planets, c, at)}
			for f := North; f <= West; f++ {
				sc.Facing = f
				for w := 20; w <= 130; w += 22 {
					for h := 6; h <= 60; h += 18 {
						l := Draw(sc, w, h)
						// 地平線の行は点と方角だけ (名前に譲らない)
						if len(l) != h || strings.Trim(l[h-1], ".NESW") != "" {
							t.Fatalf("%dx%d: %d 行 地平線 %q", w, h, len(l), l[len(l)-1])
						}
						for _, row := range l {
							if Width(row) > w {
								t.Fatalf("%s %v %v %dx%d: 幅を超えた %q", key, at, f, w, h, row)
							}
						}
						n++
					}
				}
			}
		}
	}
	if n != 4*4*4*6*4 {
		t.Fatal(n)
	}
}
