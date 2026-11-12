package sky

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

// AC-4-4: 日没の時刻の計算が、東京・2026-01-15 で公開の暦の値と 5 分以内で一致する。
// 暦の値: 国立天文台 暦計算室「日の出入り (東京) 2026 年 1 月」の 1 月 15 日 日の入 16:51
// (https://eco.mtk.nao.ac.jp/koyomi/dni/2026/s1301.html、2026-11-12 に取得。テストは通信しない)。
// 翌朝の日の出も、同じ表の 1 月 15 日の日の出 6:50 と 2 分以内で照らす (この時期の日の出は 1 日で 1 分も変わらない)。
func TestAC_4_4_SunsetMatchesAlmanac(t *testing.T) {
	c := Cities[0]
	day := time.Date(2026, 1, 15, 0, 0, 0, 0, c.Zone)
	got, ok := Sunset(c, day)
	want := time.Date(2026, 1, 15, 16, 51, 0, 0, c.Zone)
	if !ok || got.Sub(want).Abs() > 5*time.Minute {
		t.Fatalf("AC-4-4: 日没 %v (暦は %v)", got.In(c.Zone), want)
	}
	// 暦は分に丸めてあるので、本当は 1 分以内に入る
	if got.Sub(want).Abs() > time.Minute {
		t.Fatalf("AC-4-4: 日没 %v が暦から 1 分より離れている", got.In(c.Zone))
	}
	n, err := Tonight(c, day)
	if err != nil {
		t.Fatal(err)
	}
	if rise := time.Date(2026, 1, 16, 6, 50, 0, 0, c.Zone); n.Sunrise.Sub(rise).Abs() > 2*time.Minute {
		t.Fatalf("日の出 %v", n.Sunrise.In(c.Zone))
	}
	// 境界値: 時刻つきの day (同じ暦の日の 23:59) でも、同じ日の日没
	late, ok := Sunset(c, day.Add(23*time.Hour+59*time.Minute))
	if !ok || !late.Equal(got) {
		t.Fatalf("同じ日なのに日没が違う: %v %v", late, got)
	}
}

// AC-4-3: 月と惑星の角距離が 5 度以内の日時を与えると、見どころに「接近」の行が出る。
// 2026-09-08 の夜明け前 (東京) は月と木星が 1 度ほどまで近づく。
func TestAC_4_3_Approach(t *testing.T) {
	c := Cities[0]
	at := time.Date(2026, 9, 9, 3, 0, 0, 0, c.Zone)
	a := ApproachAt(Planets, c, at)
	if a == nil || a.Planet.Name != "木星" || a.Sep > ApproachDeg {
		t.Fatalf("AC-4-3: %v に月と木星の接近がない: %+v", at, a)
	}
	n, err := Tonight(c, time.Date(2026, 9, 8, 0, 0, 0, 0, c.Zone))
	if err != nil {
		t.Fatal(err)
	}
	lines := Highlights(n, 80)
	t.Log(lines)
	if len(lines) != 3 || !strings.Contains(lines[2], "月と木星が接近") || !strings.Contains(lines[2], "角距離") {
		t.Fatalf("AC-4-3: 接近の行がない %q", lines)
	}
	// 接近していない時刻 (月が木星から離れている 2 週間後) には、接近の行を出さない
	far := n
	far.Approach = ApproachAt(Planets, c, at.AddDate(0, 0, 14))
	if far.Approach != nil || len(Highlights(far, 80)) != 2 {
		t.Fatalf("AC-4-3: 離れているのに接近 %+v", far.Approach)
	}
}

// AC-4-3 の境界値: 「接近」はちょうど 5 度まで、それを少しでも超えたら接近ではない。
// 実際の空でも、03:00 (0.7 度) から月が木星を離れていき 5 度を横切る時刻を二分法で挟み、
// 直前 (5 度以内) では接近を返し、直後 (5 度を超える) では nil になる。
func TestAC_4_3_ApproachBoundary(t *testing.T) {
	if !approaching(ApproachDeg) || approaching(math.Nextafter(ApproachDeg, 6)) || !approaching(0) {
		t.Fatal("AC-4-3: 5 度ちょうどは接近、少しでも超えたら接近でない")
	}
	c := Cities[0]
	jupiter := Planets[3]
	sep := func(t time.Time) float64 {
		m := MoonAt(c, t)
		alt, az := HorizontalPlanet(jupiter, c, t)
		return Separation(m.Alt, m.Az, alt, az)
	}
	near, far := time.Date(2026, 9, 9, 3, 0, 0, 0, c.Zone), time.Date(2026, 9, 9, 15, 0, 0, 0, c.Zone)
	if sep(near) > ApproachDeg || sep(far) <= ApproachDeg {
		t.Fatalf("挟めていない: %.2f %.2f", sep(near), sep(far))
	}
	for far.Sub(near) > time.Second {
		mid := near.Add(far.Sub(near) / 2)
		if sep(mid) <= ApproachDeg {
			near = mid
		} else {
			far = mid
		}
	}
	if a := ApproachAt(Planets, c, near); a == nil || a.Planet.Name != "木星" || a.Sep > ApproachDeg {
		t.Fatalf("AC-4-3: %v (%.4f 度) で接近がない: %+v", near, sep(near), a)
	}
	if a := ApproachAt(Planets, c, far); a != nil {
		t.Fatalf("AC-4-3: %v (%.4f 度) で接近がある: %+v", far, sep(far), a)
	}
}

// 一番明るい惑星の高さの境界値: ちょうど 10 度は見える、少しでも下なら見えない。
func TestPeakAltitudeBoundary(t *testing.T) {
	if !highEnough(PeakAltitude) || highEnough(math.Nextafter(PeakAltitude, 0)) || !highEnough(90) || highEnough(-90) {
		t.Fatal("10 度ちょうどは見える、少しでも下なら見えない")
	}
	// 実際の空: 2026-01-15 (東京) の 15 分おきの時刻 1 つだけを見る brightest は、
	// 木星の高さが 10 度以上の時刻では木星を選び、10 度未満の時刻では選ばない
	c := Cities[0]
	jupiter := Planets[3]
	for h := 0; h < 24*4; h++ {
		at := time.Date(2026, 1, 15, 0, 0, 0, 0, c.Zone).Add(time.Duration(h) * 15 * time.Minute)
		alt, _ := HorizontalPlanet(jupiter, c, at)
		b := brightest([]Planet{jupiter}, c, []time.Time{at})
		if (b != nil) != highEnough(alt) {
			t.Fatalf("%v: 高さ %.2f で %+v", at, alt, b)
		}
	}
}

// AC-4-2: 見どころの行の方角は 16 方位の日本語。
func TestAC_4_2_CompassWords(t *testing.T) {
	cases := map[float64]string{
		0: "北", 11.24: "北", 11.26: "北北東", 45: "北東", 90: "東", 135: "南東", 180: "南",
		202.5: "南南西", 225: "南西", 270: "西", 315: "北西", 348.74: "北北西", 348.76: "北", 359.99: "北",
		360: "北", -45: "北西", 720 + 90: "東",
	}
	for az, want := range cases {
		if got := Compass16(az); got != want {
			t.Errorf("AC-4-2: %.2f 度は %s (want %s)", az, got, want)
		}
	}
	for _, n := range tokyoNights(t)[:30] {
		lines := Highlights(n, 80)
		if b := n.Brightest; b != nil && !strings.Contains(lines[0], Compass16(b.Az)+"の空") {
			t.Fatalf("AC-4-2: %q に方角 %s がない", lines[0], Compass16(b.Az))
		}
	}
}

// tokyoNight は東京のその日の夜。誤りならテストを止める。
func tokyoNight(t *testing.T, y int, m time.Month, d int) Night {
	t.Helper()
	c := Cities[0]
	n, err := Tonight(c, time.Date(y, m, d, 0, 0, 0, 0, c.Zone))
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// checkNarrow は、幅 w の見どころがどの行も幅に入り、惑星・接近の名前と方角、月の出の欄が残っていることを確かめる。
func checkNarrow(t *testing.T, n Night, w int) {
	t.Helper()
	lines := Highlights(n, w)
	for _, l := range lines {
		if Width(l) > w {
			t.Fatalf("幅 %d: %q", w, l)
		}
	}
	if b := n.Brightest; b != nil && (!strings.Contains(lines[0], Compass16(b.Az)) || !strings.Contains(lines[0], b.Planet.Name)) {
		t.Fatalf("幅 %d: 方角 %s が消えた %q", w, Compass16(b.Az), lines[0])
	}
	if a := n.Approach; a != nil && (!strings.Contains(lines[2], Compass16(a.Az)) || !strings.Contains(lines[2], a.Planet.Name)) {
		t.Fatalf("幅 %d: 方角 %s が消えた %q", w, Compass16(a.Az), lines[2])
	}
	if !strings.Contains(lines[1], "出") {
		t.Fatalf("幅 %d: 月の出が消えた %q", w, lines[1])
	}
}

// 狭い端末: 幅 20〜80 の全部と 1 年の全部の夜で、見どころはどの行も幅に入り、方角 (16 方位) と惑星の名前が消えない。
func TestHighlightsNarrow(t *testing.T) {
	nights := tokyoNights(t)
	for w := 20; w <= 80; w++ {
		for _, n := range nights {
			checkNarrow(t, n, w)
		}
	}
	// 幅 40 の言い方 (2026-01-15 と 2026-09-08)
	jan, sep := tokyoNight(t, 2026, 1, 15), tokyoNight(t, 2026, 9, 8)
	want := []string{"木星 -2.7 等 23:20 南 77 度", "月齢 26.4 出 04:55 入 13:23"}
	if got := Highlights(jan, 40); got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("%q", got)
	}
	if got := Highlights(sep, 40)[2]; got != "月と木星 接近 02:59 東北東 0.7 度" {
		t.Fatalf("%q", got)
	}
	if got := Highlights(sep, 20)[2]; got != "月と木星 接近 東北東" {
		t.Fatalf("%q", got)
	}
	// どの言い方も入らない幅では、いちばん短い言い方を幅で切る
	if got := fitFirst([]string{"あいうえお", "かきく"}, 4); got != "かき" {
		t.Fatalf("%q", got)
	}
}

// 2026-01-15 の東京: 日没の 1 時間後の空・一番明るい惑星は木星 (金星は太陽に近く、暗い間は地平線の下)・有明月。
func TestTonightJanuary(t *testing.T) {
	c := Cities[0]
	n, err := Tonight(c, time.Date(2026, 1, 15, 0, 0, 0, 0, c.Zone))
	if err != nil {
		t.Fatal(err)
	}
	if n.Brightest == nil || n.Brightest.Planet.Name != "木星" || n.Brightest.Alt < 70 {
		t.Fatalf("%+v", n.Brightest)
	}
	if PhaseName(n.MoonAge) != "有明月" || !n.Moonrise.OK || !n.Moonset.OK {
		t.Fatalf("%+v", n)
	}
	lines := Highlights(n, 80)
	t.Log(lines)
	if !strings.HasPrefix(lines[0], "一番明るい惑星は木星 (-2.7 等): ") || !strings.Contains(lines[1], "月齢 26.4 (有明月)  月の出 04:55  月の入り 13:23") {
		t.Fatalf("%q", lines)
	}
}

// 惑星が昇らない夜・月の出入りがない夜の文。
func TestHighlightsWithoutEvents(t *testing.T) {
	n := Night{City: Cities[0], MoonAge: 0}
	lines := Highlights(n, 80)
	if len(lines) != 2 || lines[0] != "今夜は高さ 10 度より上に昇る惑星がない" || lines[1] != "月齢 0.0 (新月)  月の出なし  月の入りなし" {
		t.Fatalf("%q", lines)
	}
	if lines := Highlights(n, 20); lines[0] != "高く昇る惑星なし" || lines[1] != "出なし 入なし" {
		t.Fatalf("%q", lines)
	}
	if brightest(nil, Cities[0], []time.Time{time.Now()}) != nil {
		t.Fatal("惑星がないのに一番明るい惑星がある")
	}
	for age, want := range map[float64]string{0: "新月", 3.7: "三日月", 7.4: "上弦", 11: "十三夜", 14.8: "満月", 18.5: "寝待月", 22.1: "下弦", 25.8: "有明月", 29.5: "新月"} {
		if got := PhaseName(age); got != want {
			t.Errorf("月齢 %.1f は %s (want %s)", age, got, want)
		}
	}
}

// 白夜・極夜の都市 (内蔵の都市にはないが、計算は止まらずに ErrNoNight を返す)。
func TestTonightPolar(t *testing.T) {
	// 時刻の帯を 2 時間進めて、南中 (13 時前) を正午より後にする (極夜の直前の短い昼でも、正午から日没を探せる)
	north := City{Key: "north", Label: "北の端", LatDeg: 78, LonDeg: 15, Zone: time.FixedZone("N", 2*60*60)}
	if _, err := Tonight(north, time.Date(2026, 6, 21, 0, 0, 0, 0, north.Zone)); !errors.Is(err, ErrNoNight) {
		t.Fatalf("白夜で %v", err)
	}
	// 極夜が始まる日: 日は沈むが、24 時間のうちに昇らない
	setsOnly := 0
	for d := range 365 {
		day := time.Date(2026, 1, 1, 0, 0, 0, 0, north.Zone).AddDate(0, 0, d)
		_, set := Sunset(north, day)
		if _, err := Tonight(north, day); set && errors.Is(err, ErrNoNight) {
			setsOnly++
		}
	}
	if setsOnly == 0 {
		t.Fatal("日が沈んで昇らない日がない")
	}
}

// 見る範囲の終わりが刻みの途中でも、そこで止めて横切りを見つける。範囲に横切りがなければ false。
func TestNextPassageEdges(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ramp := func(t time.Time) float64 { return t.Sub(from).Minutes() - 12 } // 12 分に 0 を下から上へ横切る
	got, ok := nextPassage(ramp, 0, from, from.Add(15*time.Minute), true)
	if !ok || got.Sub(from.Add(12*time.Minute)).Abs() > time.Second {
		t.Fatal(got, ok)
	}
	if _, ok := nextPassage(ramp, 0, from, from.Add(15*time.Minute), false); ok {
		t.Fatal("上から下への横切りはない")
	}
	if _, ok := nextPassage(ramp, 0, from, from.Add(11*time.Minute), true); ok {
		t.Fatal("範囲の外の横切りを見つけた")
	}
	if s := darkSamples(from, from.Add(-time.Hour)); len(s) != 1 || !s[0].Equal(from) {
		t.Fatal(s)
	}
	if s := darkSamples(from, from.Add(time.Hour)); len(s) != 7 {
		t.Fatal(len(s))
	}
}

// 等級: Meeus『天文アルゴリズム』例 41.a の金星 (1992-12-20 0 時。r = 0.7246・Δ = 0.9109 au・位相角 73.0 度) に
// 41 章の式 (この実装と同じ係数) を当てると -4.2 等。軌道要素から出した距離と位相角でも同じになる。
// 木星は 2026-01-10 の衝のころ -2.7 等ほど。
func TestMagnitude(t *testing.T) {
	venus, jupiter := Planets[1], Planets[3]
	if m := venus.Magnitude(2448976.5); math.Abs(m-(-4.2)) > 0.1 {
		t.Fatalf("金星 %.2f", m)
	}
	if m := jupiter.Magnitude(JulianDay(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))); math.Abs(m-(-2.7)) > 0.1 {
		t.Fatalf("木星 %.2f", m)
	}
}

// 太陽の位置: 2026-01-15 の東京の日の入りの方位は暦で 244.4 度。
func TestSunAzimuthAtSunset(t *testing.T) {
	c := Cities[0]
	s, ok := Sunset(c, time.Date(2026, 1, 15, 0, 0, 0, 0, c.Zone))
	if _, az := SunAt(c, s); !ok || math.Abs(az-244.4) > 0.3 {
		t.Fatalf("方位 %.2f", az)
	}
}
