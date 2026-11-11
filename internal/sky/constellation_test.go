package sky

import (
	"math"
	"strings"
	"testing"
	"time"
)

// angSep は 2 つの星の (J2000.0 の) 角距離 [度]。
func angSep(a, b Star) float64 {
	ra1, d1, ra2, d2 := a.RA*15*deg, a.Dec*deg, b.RA*15*deg, b.Dec*deg
	c := math.Sin(d1)*math.Sin(d2) + math.Cos(d1)*math.Cos(d2)*math.Cos(ra1-ra2)
	return math.Acos(math.Max(-1, math.Min(1, c))) / deg
}

// tableIndex は星表を名前で引けるようにする。同じ名前が 2 つあれば落とす。
func tableIndex(t *testing.T) map[string]Star {
	t.Helper()
	idx := map[string]Star{}
	for _, s := range Stars() {
		if _, dup := idx[s.Name]; dup {
			t.Fatalf("星表に同じ名前が 2 つ: %s", s.Name)
		}
		idx[s.Name] = s
	}
	if len(idx) != len(Catalog)+len(Faint) {
		t.Fatalf("星表 %d 個 (目録 %d + 暗い星 %d のはず)", len(idx), len(Catalog), len(Faint))
	}
	return idx
}

// AC-2-4: 埋め込んだ星座線が参照する星は、すべて埋め込んだ星表 (Stars()) にある。
func TestAC_2_4_LinesReferenceTable(t *testing.T) {
	idx := tableIndex(t)
	n := 0
	for _, c := range Constellations {
		for _, l := range c.Lines {
			for _, name := range l {
				if _, ok := idx[name]; !ok {
					t.Fatalf("AC-2-4: %s の線 %v の %s が星表にない", c.Name, l, name)
				}
				n++
			}
		}
	}
	if n < 2*60 {
		t.Fatalf("AC-2-4: 線の端 %d 個", n)
	}
}

// 星座は 12 個以上。名前は「〜座」で重ならない。線は書き写しの誤りがない (長くても 35 度・同じ星どうしは結ばない)。
func TestConstellationShapes(t *testing.T) {
	idx := tableIndex(t)
	if len(Constellations) < 12 {
		t.Fatalf("星座が %d 個 (12 個以上のはず)", len(Constellations))
	}
	names := map[string]bool{}
	for _, c := range Constellations {
		if names[c.Name] || !strings.HasSuffix(c.Name, "座") || len(c.Lines) == 0 {
			t.Fatalf("星座の名前か線がおかしい: %q", c.Name)
		}
		names[c.Name] = true
		for _, l := range c.Lines {
			if sep := angSep(idx[l[0]], idx[l[1]]); sep > 35 || l[0] == l[1] {
				t.Fatalf("%s の線 %v が %.1f 度", c.Name, l, sep)
			}
		}
	}
}

// 暗い星は線のためだけにある: どれも線に使われ、2 等級より暗く 5 等級より明るい。
func TestFaintStars(t *testing.T) {
	used := map[string]bool{}
	for _, c := range Constellations {
		for _, l := range c.Lines {
			used[l[0]], used[l[1]] = true, true
		}
	}
	for _, s := range Faint {
		if !used[s.Name] || s.Mag < 2 || s.Mag >= 5 || s.RA < 0 || s.RA >= 24 || s.Dec < -90 || s.Dec > 90 {
			t.Fatalf("暗い星がおかしい: %+v (線に使われている: %v)", s, used[s.Name])
		}
	}
}

// Stars() は呼ぶたびに新しい切り片を返す (目録を書き換えない)。
func TestStarsCopies(t *testing.T) {
	s := Stars()
	s[0].Name = "x"
	if Catalog[0].Name == "x" || Stars()[0].Name != Catalog[0].Name {
		t.Fatal("Stars() が目録を共有している")
	}
}

// AC-2-2: 星座線の両端の星が両方とも地平線より下なら、その線は描かれない (判定の境界値)。
func TestAC_2_2_SegmentShownBoundary(t *testing.T) {
	// 高度 0 は「地平線より上」でない。その上下の値とすべて組み合わせる
	alts := []float64{-90, -1e-9, 0, 1e-9, 45, 90}
	n := 0
	for _, a := range alts {
		for _, b := range alts {
			want := a > 0 || b > 0
			if got := segmentShown(Seen{Alt: a}, Seen{Alt: b}); got != want {
				t.Fatalf("AC-2-2: segmentShown(%v, %v) = %v", a, b, got)
			}
			n++
		}
	}
	if n != 36 {
		t.Fatal(n)
	}
}

// 東京からは見えない南の空の 2 つの星 (赤緯 -80 度台) と、いつも見える北の空の 2 つの星。
var (
	testTable = []Star{
		{"南1", 6, -80, 3}, {"南2", 7, -82, 3},
		{"北1", 6, 80, 3}, {"北2", 7, 82, 3},
	}
	testCons = []Constellation{
		{"みなみ座", [][2]string{{"南1", "南2"}}},
		{"かため座", [][2]string{{"北1", "南1"}}},
		{"きた座", [][2]string{{"北1", "北2"}, {"北2", "ない星"}}},
	}
)

// AC-2-2: 両端とも地平線より下の線だけの星座は Figures に出ず、図にも線の文字が出ない。
func TestAC_2_2_SegmentBelowHorizonNotShown(t *testing.T) {
	c := tokyo(t)
	figs := Figures(testCons, testTable, c, jan15)
	if len(figs) != 2 || figs[0].Name != "かため座" || figs[1].Name != "きた座" {
		t.Fatalf("AC-2-2: %+v", figs)
	}
	south := Figures(testCons[:1], testTable, c, jan15)
	out := strings.Join(Draw(Scene{Figures: south}, 80, 23), "\n")
	if len(south) != 0 || strings.ContainsAny(out, `-|/\`) || strings.Contains(out, "みなみ座") {
		t.Fatalf("AC-2-2:\n%s", out)
	}
}

// 片方だけ見える線は描く (地平線から昇ってくる星座)。点は見えている星だけ。
// 星表にない星の線は飛ばし、同じ星の点は 1 度だけ。
func TestFiguresPartial(t *testing.T) {
	figs := Figures(testCons, testTable, tokyo(t), jan15)
	if len(figs[0].Segments) != 1 || len(figs[0].Stars) != 1 || figs[0].Stars[0].Star.Name != "北1" {
		t.Fatalf("%+v", figs[0])
	}
	if len(figs[1].Segments) != 1 || len(figs[1].Stars) != 2 {
		t.Fatalf("%+v", figs[1])
	}
}

// TestSpec_SegmentShown: 4. 本物の星座で 1 年の時刻を回し、描く線はどれも少なくとも一方の端が地平線より上で、
// 描いた線と両端とも下の線を足すと全部になる (描くべき線を落とさない)。
func TestSpec_SegmentShown(t *testing.T) {
	c := tokyo(t)
	total := 0
	for _, con := range Constellations {
		total += len(con.Lines)
	}
	n := 0
	for d := 0; d < 365; d += 5 {
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(d)*24*time.Hour + time.Duration(d%24)*time.Hour)
		shown := checkFigures(t, Figures(Constellations, Stars(), c, at))
		if below := countBelow(t, c, at); shown+below != total {
			t.Fatalf("%v: 描いた %d + 下 %d != 全部 %d", at, shown, below, total)
		}
		n++
	}
	if n != 73 {
		t.Fatal(n)
	}
}

// checkFigures は、どの線も少なくとも一方の端が上・どの点も上・どの星座にも点があることを確かめ、線の数を返す。
func checkFigures(t *testing.T, figs []Figure) int {
	t.Helper()
	shown := 0
	for _, f := range figs {
		if len(f.Stars) == 0 {
			t.Fatalf("%s に見える星がない", f.Name)
		}
		for _, s := range f.Segments {
			if s.A.Alt <= 0 && s.B.Alt <= 0 {
				t.Fatalf("%s の線 %s-%s は両端とも地平線の下", f.Name, s.A.Star.Name, s.B.Star.Name)
			}
			shown++
		}
		for _, s := range f.Stars {
			if s.Alt <= 0 {
				t.Fatalf("点 %s が地平線の下", s.Star.Name)
			}
		}
	}
	return shown
}

// countBelow は両端とも地平線より下の線の数。
func countBelow(t *testing.T, c City, at time.Time) int {
	t.Helper()
	idx := tableIndex(t)
	below := 0
	for _, con := range Constellations {
		for _, l := range con.Lines {
			a, _ := Horizontal(idx[l[0]], c, at)
			b, _ := Horizontal(idx[l[1]], c, at)
			if a <= 0 && b <= 0 {
				below++
			}
		}
	}
	return below
}
