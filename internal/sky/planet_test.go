package sky

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

func planet(t *testing.T, name string) Planet {
	t.Helper()
	for _, p := range Planets {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("%s がない", name)
	return Planet{}
}

// AC-3-1: 2026-01-15 の木星の赤経・赤緯が、公開の暦の値と 1 度以内で一致する。
// 比べる値の出典: NASA JPL Horizons (ssd.jpl.nasa.gov/api/horizons.api, COMMAND=599, CENTER=500@399 (地球の中心),
// 2026-01-15 00:00 UT, QUANTITIES=1 の ICRF の位置)。2026-11-12 に取得: 赤経 110.73180 度・赤緯 22.34274 度。
// テストは通信しない (値をここに書き写した)。
func TestAC_3_1_JupiterRADec(t *testing.T) {
	jd := JulianDay(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	ra, dec := planet(t, "木星").Equatorial(jd)
	t.Logf("木星 赤経 %.4f 赤緯 %.4f", ra, dec)
	if angDiff(ra, 110.73180) > 1 || math.Abs(dec-22.34274) > 1 {
		t.Fatalf("AC-3-1: 木星の赤経 %.3f 赤緯 %.3f", ra, dec)
	}
	// 精度の目安: 近似の要素でも 0.02 度より近い (2026-11-12 の実測は 0.003 度)
	if angDiff(ra, 110.73180) > 0.02 || math.Abs(dec-22.34274) > 0.02 {
		t.Fatalf("木星の誤差が大きい: 赤経 %.3f 赤緯 %.3f", ra, dec)
	}
}

// AC-3-2: 2026-01-15 前後の新月・満月の日付で、計算した月齢が新月は 0〜1.5 日・満月は 13.5〜16 日に入る。
// 新月・満月の時刻の出典: 米国海軍天文台 (USNO) の公開の暦 (aa.usno.navy.mil/api/moon/phases/date, date=2025-12-15, nump=6)。
// 2026-11-12 に取得: 新月 2025-12-20 01:43 UT・満月 2026-01-03 10:03 UT・新月 2026-01-18 19:52 UT。
// その日付 (日本時間) の 21 時 (sky now を打つ時刻) の月齢を見る。
func TestAC_3_2_MoonAgeAtPhases(t *testing.T) {
	cases := []struct {
		at       time.Time
		min, max float64
	}{
		{time.Date(2025, 12, 20, 21, 0, 0, 0, jst), 0, 1.5}, // 新月
		{time.Date(2026, 1, 3, 21, 0, 0, 0, jst), 13.5, 16}, // 満月
		{time.Date(2026, 1, 19, 21, 0, 0, 0, jst), 0, 1.5},  // 新月 (01-18 19:52 UT = 01-19 04:52 JST)
	}
	for _, c := range cases {
		age := MoonAge(JulianDay(c.at))
		t.Logf("%v 月齢 %.2f", c.at, age)
		if age < c.min || age > c.max {
			t.Fatalf("AC-3-2: %v の月齢 %.2f (%v〜%v)", c.at, age, c.min, c.max)
		}
	}
	// 新月・満月の時刻そのものの離角は、0 度・180 度から 1 度以内 (簡易式の誤差の目安)
	for _, c := range []struct {
		at   time.Time
		want float64
	}{
		{time.Date(2025, 12, 20, 1, 43, 0, 0, time.UTC), 0},
		{time.Date(2026, 1, 3, 10, 3, 0, 0, time.UTC), 180},
		{time.Date(2026, 1, 18, 19, 52, 0, 0, time.UTC), 0},
	} {
		jd := JulianDay(c.at)
		lon, _, _ := MoonEcliptic(jd)
		if d := angDiff(lon-SunLongitude(jd), c.want); d > 1 {
			t.Fatalf("%v: 離角のずれ %.2f 度", c.at, d)
		}
	}
}

// 月の高度・方位は、公開の暦と 0.1 度以内 (2026-11-12 の実測は高度 0.05 度・方位 0.01 度)。
// 出典: NASA JPL Horizons (COMMAND=301, 東京の都市の中心 東経 139.6917・北緯 35.6895 から、2026-01-03 12:00 UT、
// QUANTITIES=4、APPARENT=AIRLESS (大気の屈折なし))。2026-11-12 に取得: 方位 91.168007 度・高度 51.777795 度。
func TestMoonAt_Tokyo(t *testing.T) {
	m := MoonAt(tokyo(t), time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC))
	t.Logf("%+v", m)
	if math.Abs(m.Alt-51.777795) > 0.1 || angDiff(m.Az, 91.168007) > 0.1 || !m.Up() {
		t.Fatalf("月 %+v", m)
	}
}

func TestSunLongitude(t *testing.T) {
	// Meeus 例 25.a: 1992-10-13 0h TD の太陽の真黄経 199.90988 度 (簡易式は 0.01 度ほど)
	if got := SunLongitude(2448908.5); math.Abs(got-199.90988) > 0.02 {
		t.Fatal(got)
	}
}

func TestMoonEcliptic(t *testing.T) {
	// Meeus 例 47.a: 1992-04-12 0h TD → 黄経 133.162655・黄緯 -3.229126・距離 368409.7 km (大きい項だけなので 0.3 度・1000 km)
	lon, lat, dist := MoonEcliptic(2448724.5)
	if angDiff(lon, 133.162655) > 0.3 || math.Abs(lat+3.229126) > 0.3 || math.Abs(dist-368409.7) > 1000 {
		t.Fatal(lon, lat, dist)
	}
}

func TestMoonSymbol(t *testing.T) {
	for _, c := range []struct {
		age  float64
		want rune
	}{{0, 'o'}, {3, ')'}, {7.4, 'D'}, {11, 'O'}, {14.8, 'O'}, {18.5, 'O'}, {22.1, 'C'}, {26, '('}, {29.4, 'o'}} {
		if got := MoonSymbol(c.age); got != c.want {
			t.Fatalf("月齢 %v: %c, want %c", c.age, got, c.want)
		}
	}
	// 記号は星・星座線・地平線の文字と重ならない
	for _, r := range moonSymbols {
		if strings.ContainsRune("●*+-|/\\.NSEW", r) || RuneWidth(r) != 1 {
			t.Fatalf("月の記号 %c", r)
		}
	}
}

func TestPlanets(t *testing.T) {
	names, syms := map[string]bool{}, map[rune]bool{}
	for _, p := range Planets {
		if names[p.Name] || syms[p.Symbol] || RuneWidth(p.Symbol) != 2 || !strings.HasPrefix(p.Name, string(p.Symbol)) {
			t.Fatalf("%+v", p)
		}
		names[p.Name], syms[p.Symbol] = true, true
	}
	if len(Planets) != 5 {
		t.Fatal(len(Planets))
	}
	// 太陽から地球までは約 1 au (0.983〜1.017)
	for d := 0; d < 365; d += 7 {
		x, y, z := earthOrbit.heliocentric(j2000 + float64(d))
		if r := math.Sqrt(x*x + y*y + z*z); r < 0.98 || r > 1.02 {
			t.Fatalf("%d 日: %v au", d, r)
		}
	}
	// ほかの 4 つも、AC-3-1 と同じ JPL Horizons の 2026-01-15 00:00 UT の値 (2026-11-12 に取得) と 1 度以内。
	// 水星・金星は太陽に近い (この日は合のころ) ので、地球の位置の誤差も効くが、それでも 1 度に入る
	jd := JulianDay(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	for _, c := range []struct {
		name    string
		ra, dec float64
	}{{"水星", 292.21596, -23.66999}, {"金星", 298.67640, -21.81972}, {"火星", 295.13105, -22.41230}, {"土星", 357.91315, -3.31933}} {
		ra, dec := planet(t, c.name).Equatorial(jd)
		t.Logf("%s 赤経 %.4f 赤緯 %.4f", c.name, ra, dec)
		if angDiff(ra, c.ra) > 1 || math.Abs(dec-c.dec) > 1 {
			t.Fatalf("%s: %v %v", c.name, ra, dec)
		}
	}
}

func TestVisiblePlanets(t *testing.T) {
	// 2026-01-15 21:00 の東京: 木星は衝のころで高く東の空、金星は太陽の向こう側 (外合のころ) で沈んでいる
	seen := VisiblePlanets(Planets, tokyo(t), jan15)
	var jupiter, venus bool
	for _, s := range seen {
		jupiter = jupiter || s.Planet.Name == "木星"
		venus = venus || s.Planet.Name == "金星"
	}
	if !jupiter || venus {
		t.Fatalf("%+v", seen)
	}
}

// 境界値: 高度ちょうど 0 は地平線の上ではない (月も惑星も描かない・凡例に出さない)。
func TestHorizonBoundary(t *testing.T) {
	if (MoonSeen{Alt: 0}).Up() || !(MoonSeen{Alt: 1e-9}).Up() || (MoonSeen{Alt: -1e-9}).Up() {
		t.Fatal("月の地平線の境目")
	}
	jup := planet(t, "木星")
	if got := Legend(MoonSeen{Alt: 0, Age: 3}, nil, 80); got != NothingUp {
		t.Fatal(got)
	}
	// 高度 0 の月は図にも描かない
	if strings.ContainsRune(strings.Join(Draw(Scene{Moon: MoonSeen{Alt: 0, Az: 90, Age: 3}}, 80, 22), ""), ')') {
		t.Fatal("高度 0 の月を描いた")
	}
	// VisiblePlanets は高度 0 を含めない: 木星の高度が 0 を横切る時刻を 1 秒刻みで挟み、境目の前後で入る・入らないが切り替わる
	c := tokyo(t)
	lo, hi := jan15, jan15.Add(12*time.Hour) // 21 時は上、翌 9 時は下
	if a, _ := HorizontalPlanet(jup, c, lo); a <= 0 {
		t.Fatal(a)
	}
	if a, _ := HorizontalPlanet(jup, c, hi); a > 0 {
		t.Fatal(a)
	}
	for hi.Sub(lo) > time.Second {
		mid := lo.Add(hi.Sub(lo) / 2)
		if a, _ := HorizontalPlanet(jup, c, mid); a > 0 {
			lo = mid
		} else {
			hi = mid
		}
	}
	has := func(at time.Time) bool {
		seen := VisiblePlanets([]Planet{jup}, c, at)
		if len(seen) == 1 && seen[0].Alt <= 0 {
			t.Fatalf("%v: 高度 %v", at, seen[0].Alt)
		}
		return len(seen) == 1
	}
	if !has(lo) || has(hi) || strings.Contains(Legend(MoonSeen{}, VisiblePlanets([]Planet{jup}, c, hi), 80), "木星") {
		t.Fatalf("境目 %v〜%v", lo, hi)
	}
}

func TestLegend(t *testing.T) {
	if got := Legend(MoonSeen{Alt: -3, Age: 5}, nil, 20); got != NothingUp || Width(NothingUp) > 20 {
		t.Fatal(got)
	}
	ps := []PlanetSeen{{Planet: planet(t, "木星"), Alt: 10}, {Planet: planet(t, "土星"), Alt: 5}}
	moon := MoonSeen{Alt: 20, Age: 7.43}
	// 幅に合わせて 3 段で縮める (それぞれの段のちょうどの幅と、1 つ狭い幅)
	for _, c := range []struct {
		width int
		want  string
	}{
		{80, "D 月 (月齢 7.4)  木 木星  土 土星"},
		{33, "D 月 (月齢 7.4)  木 木星  土 土星"},
		{32, "D 月齢 7.4 木星 土星"},
		{20, "D 月齢 7.4 木星 土星"},
		{19, "D 月齢 7.4 ほか 2"},
	} {
		if got := Legend(moon, ps, c.width); got != c.want {
			t.Fatalf("幅 %d: %q, want %q", c.width, got, c.want)
		}
	}
	if got := Legend(MoonSeen{Alt: -1}, ps[:1], 80); got != "木 木星" {
		t.Fatal(got)
	}
	if got := fitWithRest([]string{"D 月齢 14.8", "水星", "金星"}, 9); got != "ほか 3" {
		t.Fatal(got)
	}
}

// 全件: 月 (上・下 × 8 つの相) × 惑星の組 (2^5) × 幅 20〜120 で、凡例は幅に収まり、線の文字を使わず、
// 地平線より上のものは名前で出るか「ほか <数>」に数えられる (黙って消えない)。地平線の下のものは出ない。
func TestLegendAllFits(t *testing.T) {
	n := 0
	for _, alt := range []float64{-1, 1} {
		for ph := 0; ph < 8; ph++ {
			moon := MoonSeen{Alt: alt, Age: float64(ph)*SynodicMonth/8 + 0.3}
			for mask := 0; mask < 1<<len(Planets); mask++ {
				var ps []PlanetSeen
				for i, p := range Planets {
					if mask&(1<<i) != 0 {
						ps = append(ps, PlanetSeen{Planet: p, Alt: 10})
					}
				}
				for w := 20; w <= 120; w++ {
					checkLegend(t, Legend(moon, ps, w), moon, ps, w)
					n++
				}
			}
		}
	}
	if n != 2*8*32*101 {
		t.Fatal(n)
	}
}

func checkLegend(t *testing.T, got string, moon MoonSeen, ps []PlanetSeen, w int) {
	t.Helper()
	if Width(got) > w || strings.ContainsAny(got, `-|/\`) {
		t.Fatalf("幅 %d: %q", w, got)
	}
	shown, up := 0, len(ps)
	for _, p := range ps {
		if strings.Contains(got, p.Planet.Name) {
			shown++
		}
	}
	if moon.Up() {
		up++
		if strings.Contains(got, "月齢") {
			shown++
		}
	} else if strings.Contains(got, "月齢") {
		t.Fatalf("地平線の下の月: %q", got)
	}
	rest := 0
	if i := strings.Index(got, "ほか "); i >= 0 {
		if _, err := fmt.Sscanf(got[i:], "ほか %d", &rest); err != nil {
			t.Fatal(err)
		}
	}
	if up == 0 && got != NothingUp || up > 0 && shown+rest != up {
		t.Fatalf("幅 %d: %q (上に %d、名前 %d、ほか %d)", w, got, up, shown, rest)
	}
}

// runeAt は図の行 row の、格子の列 col (全角は 2 列と数える) から始まる文字。
func runeAt(lines []string, row, col int) rune {
	n := 0
	for _, r := range lines[row] {
		if n == col {
			return r
		}
		n += RuneWidth(r)
	}
	return ' '
}

func TestDrawBodies(t *testing.T) {
	const w, h = 80, 22
	rx, ry, cx, cy := layout(w, h)
	jup := planet(t, "木星")
	// 天頂の月と、東の地平線近くの木星
	sc := Scene{Moon: MoonSeen{Alt: 90, Age: 7.4}, Planets: []PlanetSeen{{Planet: jup, Alt: 30, Az: 90}}}
	lines := Draw(sc, w, h)
	if runeAt(lines, cy, cx) != 'D' {
		t.Fatalf("天頂に月がない\n%s", strings.Join(lines, "\n"))
	}
	row, col := project(30, 90, rx, ry, cx, cy)
	if runeAt(lines, row, col) != '木' {
		t.Fatalf("木星がない\n%s", strings.Join(lines, "\n"))
	}
	// 地平線より下の月は描かない
	if strings.ContainsRune(strings.Join(Draw(Scene{Moon: MoonSeen{Alt: -5, Az: 0, Age: 7.4}}, w, h), ""), 'D') {
		t.Fatal("地平線の下の月を描いた")
	}
	// 同じ升に来た天体は、近い順 (1 つ右・2 つ左) に並べる。全部埋まれば置かない。
	// 星は天体の升 (全角の右半分も) に重ねない
	g := newGrid(w, h)
	v := newView(w, h)
	for _, r := range []rune{'o', '木', '土', '火'} {
		g.body(v, 90, 0, r)
	}
	g.stars(v, []Seen{{Star: Star{Name: "x", Mag: 0}, Alt: 90}})
	want := []cell{{'土', solid}, {0, cont}, {'o', solid}, {'木', solid}, {0, cont}}
	for i, c := range want {
		if got := g.cells[cy][cx-2+i]; got != c {
			t.Fatalf("列 %d: %+v, want %+v (%q)", cx-2+i, got, c, g.lines()[cy])
		}
	}
	if strings.ContainsAny(g.lines()[cy], "火●") {
		t.Fatalf("重ねて描いた: %q", g.lines()[cy])
	}
	// 右の端 (西の地平線。21 列では列 20 = 幅 - 1) の全角の記号は、はみ出さずに左へ
	if _, _, cx, _ := layout(21, 23); cx+10 != 20 {
		t.Fatal("21 × 23 の西の端は列 20")
	}
	lines = Draw(Scene{Planets: []PlanetSeen{{Planet: jup, Alt: 0.01, Az: 270}}}, 21, 23)
	for _, l := range lines {
		if Width(l) > 21 {
			t.Fatalf("はみ出した: %q", l)
		}
	}
	if !strings.ContainsRune(strings.Join(lines, ""), '木') {
		t.Fatalf("右の端の木星がない\n%s", strings.Join(lines, "\n"))
	}
}

func TestVacant(t *testing.T) {
	g := newGrid(10, 5)
	for _, c := range [][3]int{{-1, 0, 1}, {5, 0, 1}, {0, -1, 1}, {0, 9, 2}} {
		if g.vacant(c[0], c[1], c[2]) {
			t.Fatalf("%v は格子の外", c)
		}
	}
	if !g.vacant(0, 8, 2) {
		t.Fatal("右の端の 2 升")
	}
}
