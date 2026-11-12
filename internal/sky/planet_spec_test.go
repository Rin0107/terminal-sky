package sky

import (
	"math"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// 仕様 spec/sky.smt2 の 7・8 と、惑星の計算の性質を、実装に対して全件で確かめる。

// TestSpec_MoonAge: 7. 月齢は 0 以上・朔望月未満で、相は 0〜7 (新月 0・満月 4)。
func TestSpec_MoonAge(t *testing.T) {
	n := 0
	prev := -1.0
	for i := 0; i < 36000; i++ {
		e := float64(i) / 100
		age := ageFromElongation(e)
		p := moonPhase(age)
		if age < 0 || age >= SynodicMonth || p < 0 || p > 7 || age <= prev {
			t.Fatalf("離角 %v: 月齢 %v 相 %d", e, age, p)
		}
		prev = age
		n++
	}
	if n != 36000 {
		t.Fatal(n)
	}
	// 新月は 0、満月は 4。離角が負や 360 以上でも同じ周期で折り返す
	for _, c := range []struct {
		e    float64
		want int
	}{{0, 0}, {180, 4}, {90, 2}, {270, 6}, {-1, 0}, {360, 0}, {540, 4}, {22.4, 0}, {22.6, 1}} {
		if got := moonPhase(ageFromElongation(c.e)); got != c.want {
			t.Fatalf("離角 %v: 相 %d, want %d", c.e, got, c.want)
		}
	}
	// 朔望月の外の月齢でも相は 0〜7
	for _, age := range []float64{-0.1, -SynodicMonth, SynodicMonth, 3 * SynodicMonth, -1e9, 1e9} {
		if p := moonPhase(age); p < 0 || p > 7 {
			t.Fatalf("月齢 %v: 相 %d", age, p)
		}
	}
}

// TestSpec_BodyFits: 8. 幅 1・2 の記号は、どの列 (0〜w-1) でも、その列か 1 つ左に格子からはみ出さずに置ける
// (空の格子で、実装の判定 vacant がはじめの 2 つのどちらかで通る)。
func TestSpec_BodyFits(t *testing.T) {
	n := 0
	for w := 20; w <= 400; w++ {
		g := newGrid(w, 1)
		for col := 0; col < w; col++ {
			for _, width := range []int{1, 2} {
				ok := false
				tries := bodyColumns(col)
				for _, c := range tries[:2] {
					ok = ok || g.vacant(0, c, width)
				}
				if !ok {
					t.Fatalf("w %d col %d 幅 %d", w, col, width)
				}
				n++
			}
		}
	}
	if want := 2 * (20 + 400) * 381 / 2; n != want {
		t.Fatal(n, want)
	}
}

// TestSpec_PlanetsAboveHorizon: 描く惑星は地平線より上だけで、地平線より上の惑星は全部、凡例に名前が出る。
func TestSpec_PlanetsAboveHorizon(t *testing.T) {
	c := tokyo(t)
	n, up := 0, 0
	for d := 0; d < 365; d += 3 {
		at := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Add(time.Duration(d)*24*time.Hour + time.Duration(d%24)*time.Hour)
		seen := VisiblePlanets(Planets, c, at)
		legend := Legend(MoonSeen{}, seen, 80)
		got := map[string]bool{}
		for _, s := range seen {
			got[s.Planet.Name] = true
			if s.Alt <= 0 || !strings.Contains(legend, s.Planet.Name) {
				t.Fatalf("%v: %+v / %q", at, s, legend)
			}
		}
		for _, p := range Planets {
			alt, _ := HorizontalPlanet(p, c, at)
			if (alt > 0) != got[p.Name] {
				t.Fatalf("%v: %s 高度 %v", at, p.Name, alt)
			}
			if alt <= 0 && strings.Contains(legend, p.Name) {
				t.Fatalf("%v: 地平線の下の %s が凡例にある", at, p.Name)
			}
			n++
		}
		up += len(seen)
	}
	if n != 122*len(Planets) || up == 0 || up == n {
		t.Fatal(n, up)
	}
}

func keplerResidual(m, e float64) float64 {
	ea := solveKepler(m, e)
	return math.Abs(ea - e*math.Sin(ea) - m)
}

// TestSpec_Kepler: ケプラーの方程式の解は、離心率 0〜0.25 (水星は 0.21) と平均近点角 0〜2π の全部の格子で 1e-12 以内。
func TestSpec_Kepler(t *testing.T) {
	n := 0
	for i := 0; i <= 25; i++ {
		for j := 0; j < 720; j++ {
			e, m := float64(i)/100, float64(j)/720*2*math.Pi
			if r := keplerResidual(m, e); r > 1e-12 {
				t.Fatalf("e %v M %v: 残り %v", e, m, r)
			}
			n++
		}
	}
	if n != 26*720 {
		t.Fatal(n)
	}
	// PBT: 任意の離心率 [0, 0.25]・平均近点角 [0, 2π)
	prop := func(a, b uint32) bool {
		e := float64(a) / math.MaxUint32 * 0.25
		m := float64(b) / (math.MaxUint32 + 1.0) * 2 * math.Pi
		return keplerResidual(m, e) <= 1e-12
	}
	if err := quick.Check(prop, &quick.Config{MaxCount: 5000}); err != nil {
		t.Fatal(err)
	}
}

func FuzzMoonAge(f *testing.F) {
	f.Add(2461044.0)
	f.Add(j2000)
	f.Add(-1e7)
	f.Fuzz(func(t *testing.T, jd float64) {
		if math.IsNaN(jd) || math.IsInf(jd, 0) || math.Abs(jd-j2000) > 36525*10 {
			return
		}
		age := MoonAge(jd)
		if age < 0 || age >= SynodicMonth || !strings.ContainsRune("o)DOC(", MoonSymbol(age)) {
			t.Fatalf("jd %v: 月齢 %v", jd, age)
		}
	})
}
