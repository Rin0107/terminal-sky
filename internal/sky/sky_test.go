package sky

import (
	"math"
	"testing"
	"time"
)

func tokyo(t *testing.T) City {
	t.Helper()
	c, ok := FindCity("tokyo")
	if !ok {
		t.Fatal("tokyo がない")
	}
	return c
}

func star(t *testing.T, name string) Star {
	t.Helper()
	for _, s := range Catalog {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("%s が目録にない", name)
	return Star{}
}

func angDiff(a, b float64) float64 {
	d := math.Abs(norm360(a) - norm360(b))
	return math.Min(d, 360-d)
}

var jan15 = time.Date(2026, 1, 15, 21, 0, 0, 0, jst)

// AC-1-2: 東京・2026-01-15T21:00+09:00 のシリウスの高度と方位が、暦の値と 1 度以内。
// 比べる値の出典: 米国海軍天文台 (USNO) の公開の暦 Celestial Navigation Data
// (aa.usno.navy.mil/api/celnav, date=2026-01-15, time=12:00 UT, 東京の都市の中心 北緯 35.6895・東経 139.6917)。
// 2026-11-10 に取得: Sirius の Hc (高度) 31.681191 度・Zn (方位) 149.355634 度。
// テストは通信しない (値をここに書き写した)。南中高度とも照らす (TestHorizontal_TransitAltitude)。
func TestAC_1_2_SiriusTokyo(t *testing.T) {
	alt, az := Horizontal(star(t, "シリウス"), tokyo(t), jan15)
	t.Logf("シリウス 高度 %.3f 方位 %.3f", alt, az)
	if math.Abs(alt-31.681191) > 1 || angDiff(az, 149.355634) > 1 {
		t.Fatalf("AC-1-2: シリウスの高度 %.2f 方位 %.2f", alt, az)
	}
}

func TestHorizontal_TransitAltitude(t *testing.T) {
	c := tokyo(t)
	s := star(t, "シリウス")
	// 1 分刻みで 1 日分を回し、一番高いときの高度と方位を見る。
	best, bestAz, n := -100.0, 0.0, 0
	for m := 0; m < 24*60; m++ {
		alt, az := Horizontal(s, c, jan15.Add(time.Duration(m)*time.Minute))
		if alt > best {
			best, bestAz = alt, az
		}
		n++
	}
	if n != 1440 {
		t.Fatal(n)
	}
	_, dec := Precess(s.RA*15, s.Dec, JulianDay(jan15))
	if want := 90 - c.LatDeg + dec; math.Abs(best-want) > 0.05 {
		t.Fatalf("南中高度 %.3f, 期待 %.3f", best, want)
	}
	if angDiff(bestAz, 180) > 1 {
		t.Fatalf("南中の方位 %.2f", bestAz)
	}
}

func TestHorizontal_PolarisNearLatitude(t *testing.T) {
	c := tokyo(t)
	for h := 0; h < 24; h++ {
		alt, az := Horizontal(star(t, "ポラリス"), c, jan15.Add(time.Duration(h)*time.Hour))
		if math.Abs(alt-c.LatDeg) > 1 || angDiff(az, 0) > 1.5 {
			t.Fatalf("%d 時: ポラリス 高度 %.2f 方位 %.2f", h, alt, az)
		}
	}
}

func TestHorizontal_Ranges(t *testing.T) {
	c := tokyo(t)
	n := 0
	for _, s := range Catalog {
		for h := 0; h < 48; h++ {
			alt, az := Horizontal(s, c, jan15.Add(time.Duration(h)*30*time.Minute))
			if alt < -90 || alt > 90 || az < 0 || az >= 360 || math.IsNaN(alt) || math.IsNaN(az) {
				t.Fatalf("%s: 高度 %v 方位 %v", s.Name, alt, az)
			}
			n++
		}
	}
	if n != len(Catalog)*48 {
		t.Fatal(n)
	}
}

func TestJulianDay(t *testing.T) {
	if got := JulianDay(time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)); got != j2000 {
		t.Fatalf("J2000.0 = %v", got)
	}
	// Meeus 例 7.a: 1957-10-04 19:26:24 UT = JD 2436116.31
	if got := JulianDay(time.Date(1957, 10, 4, 19, 26, 24, 0, time.UTC)); math.Abs(got-2436116.31) > 1e-6 {
		t.Fatalf("got %v", got)
	}
}

func TestGMST(t *testing.T) {
	// Meeus 例 12.a: 1987-04-10 0h UT の平均恒星時 13h10m46.3668s
	jd := JulianDay(time.Date(1987, 4, 10, 0, 0, 0, 0, time.UTC))
	if got, want := GMST(jd), hms(13, 10, 46.3668)*15; math.Abs(got-want) > 1e-5 {
		t.Fatalf("got %v want %v", got, want)
	}
	// Meeus 例 12.b: 1987-04-10 19:21:00 UT → 8h34m57.0896s
	jd = JulianDay(time.Date(1987, 4, 10, 19, 21, 0, 0, time.UTC))
	if got, want := GMST(jd), hms(8, 34, 57.0896)*15; math.Abs(got-want) > 1e-4 {
		t.Fatalf("got %v want %v", got, want)
	}
}

// UnixNano があふれる時刻 (1678 年より前・2262 年より後) でも正しいユリウス日になる。
func TestJulianDayFarDates(t *testing.T) {
	// 2400-01-01 0h UT = J2000.0 + 400 年 (グレゴリオ暦の 146097 日) - 0.5 日
	if got := JulianDay(time.Date(2400, 1, 1, 0, 0, 0, 0, time.UTC)); got != 2597641.5 {
		t.Fatalf("2400 年: %v", got)
	}
	// 1600-01-01 0h UT = 2000-01-01 0h UT - 146097 日
	if got := JulianDay(time.Date(1600, 1, 1, 0, 0, 0, 0, time.UTC)); got != 2451544.5-146097 {
		t.Fatalf("1600 年: %v", got)
	}
	// 秒より細かい部分も足す (12 時間 = 0.5 日 のうち、半分を秒の下で)
	if got := JulianDay(time.Date(2000, 1, 1, 0, 0, 0, 500_000_000, time.UTC)); math.Abs(got-(2451544.5+0.5/86400)) > 1e-9 {
		t.Fatalf("ナノ秒: %v", got)
	}
}

func TestPrecess(t *testing.T) {
	// Meeus 例 21.b: θ Persei (固有運動を足したあとの J2000.0 の位置 赤経 41.054063°・赤緯 49.227750°)
	// → 2028-11-13.19 TD (JD 2462088.69) の平均位置 赤経 41.547214°・赤緯 49.348483°
	ra, dec := Precess(41.054063, 49.227750, 2462088.69)
	if math.Abs(ra-41.547214) > 1e-4 || math.Abs(dec-49.348483) > 1e-4 {
		t.Fatalf("ra %.6f dec %.6f", ra, dec)
	}
	// J2000.0 ではそのまま
	ra, dec = Precess(100, -20, j2000)
	if math.Abs(ra-100) > 1e-9 || math.Abs(dec+20) > 1e-9 {
		t.Fatalf("ra %v dec %v", ra, dec)
	}
}

func TestNorm360(t *testing.T) {
	for _, c := range [][2]float64{{0, 0}, {360, 0}, {-1, 359}, {361, 1}, {-360, 0}, {720.5, 0.5}, {-1e-17, 0}} {
		if got := norm360(c[0]); math.Abs(got-c[1]) > 1e-9 {
			t.Fatalf("norm360(%v) = %v", c[0], got)
		}
	}
}

func TestHmsDms(t *testing.T) {
	if hms(6, 30, 36) != 6.51 {
		t.Fatal(hms(6, 30, 36))
	}
	if got := dms(-16, 30, 36); math.Abs(got+16.51) > 1e-12 {
		t.Fatal(got)
	}
	if got := dms(16, 30, 36); math.Abs(got-16.51) > 1e-12 {
		t.Fatal(got)
	}
	if got := dms(0, 30, 0); got != 0.5 {
		t.Fatal(got)
	}
}

func TestCatalog(t *testing.T) {
	if len(Catalog) < 45 || len(Catalog) > 55 {
		t.Fatalf("目録の数 %d (約 50 個)", len(Catalog))
	}
	names := map[string]bool{}
	for _, s := range Catalog {
		if s.Mag >= 2 || s.RA < 0 || s.RA >= 24 || s.Dec < -90 || s.Dec > 90 || s.Name == "" || names[s.Name] {
			t.Fatalf("目録の星がおかしい: %+v", s)
		}
		names[s.Name] = true
	}
}

func TestFindCity(t *testing.T) {
	for _, k := range []string{"tokyo", "Tokyo", " TOKYO "} {
		if c, ok := FindCity(k); !ok || c.Label != "東京" {
			t.Fatalf("%q: %v %v", k, c, ok)
		}
	}
	if _, ok := FindCity("atlantis"); ok {
		t.Fatal("ない都市が見つかった")
	}
	// 一覧の順は Cities の順で、既定の東京が先頭
	if got := CityKeys(); len(got) != len(Cities) || got[0] != "tokyo" || got[len(got)-1] != Cities[len(Cities)-1].Key {
		t.Fatal(got)
	}
	// 表示の名前でも選べる
	if c, ok := FindCity("札幌"); !ok || c.Key != "sapporo" {
		t.Fatal(c, ok)
	}
}

func TestVisible(t *testing.T) {
	seen := Visible(Catalog, tokyo(t), jan15)
	if len(seen) == 0 || len(seen) >= len(Catalog) {
		t.Fatalf("見える星 %d", len(seen))
	}
	for i, s := range seen {
		if s.Alt <= 0 {
			t.Fatalf("地平線の下: %+v", s)
		}
		if i > 0 && seen[i-1].Star.Mag > s.Star.Mag {
			t.Fatal("明るい順でない")
		}
	}
	// 同じ等級は目録の順 (安定な並べ替え)
	same := []Star{{"b", 6, 20, 1}, {"a", 6, 20, 1}}
	if got := Visible(same, tokyo(t), jan15); len(got) != 2 || got[0].Star.Name != "b" {
		t.Fatal(got)
	}
	// 南十字は東京から見えない
	for _, s := range seen {
		if s.Star.Name == "アクルックス" {
			t.Fatal("アクルックスが見えている")
		}
	}
}
