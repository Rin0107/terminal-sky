package sky

import (
	"math"
	"testing"
	"time"
)

// TestSpec_Compass16 は spec/sky.smt2 の 9 を実装で確かめる: 方位 -720〜720 度の 0.01 度刻みの全部で、
// 16 方位の番号は 0〜15 で、その方位の中心 (番号・22.5 度) からのずれは 11.25 度まで (360 度で折り返して測る)。
func TestSpec_Compass16(t *testing.T) {
	for i := -72000; i <= 72000; i++ {
		az := float64(i) / 100
		k := compassIndex(az)
		if k < 0 || k > 15 {
			t.Fatalf("方位 %.2f の番号 %d", az, k)
		}
		d := math.Abs(norm360(az) - float64(k)*22.5)
		d = math.Min(d, 360-d)
		if d > 11.25+1e-9 {
			t.Fatalf("方位 %.2f は %s (%d) から %.2f 度ずれている", az, compassPoints[k], k, d)
		}
		if Compass16(az) != compassPoints[k] {
			t.Fatal(az)
		}
	}
}

// tokyoNights は東京の 2026 年の全部の夜。
func tokyoNights(t *testing.T) []Night {
	t.Helper()
	c := Cities[0]
	var out []Night
	for d := range 365 {
		day := time.Date(2026, 1, 1, 0, 0, 0, 0, c.Zone).AddDate(0, 0, d)
		n, err := Tonight(c, day)
		if err != nil {
			t.Fatalf("%v: %v", day, err)
		}
		out = append(out, n)
	}
	return out
}

// TestSpec_HighlightLines は spec/sky.smt2 の 10 を実装で確かめる: 東京の 1 年の全部の夜で、見どころは 1〜3 行で、
// 接近があるときだけ 3 行。どの行も空でない。
func TestSpec_HighlightLines(t *testing.T) {
	approaches := 0
	for _, n := range tokyoNights(t) {
		lines := Highlights(n, 400)
		if len(lines) < 1 || len(lines) > 3 || (len(lines) == 3) != (n.Approach != nil) {
			t.Fatalf("%v: %d 行 %q", n.Sunset, len(lines), lines)
		}
		for _, l := range lines {
			if l == "" {
				t.Fatalf("%v: 空の行", n.Sunset)
			}
		}
		if n.Approach != nil {
			approaches++
		}
	}
	if approaches == 0 {
		t.Fatal("1 年に 1 度も接近がない")
	}
}

// TestSpec_SunsetCrossing: 東京の 1 年の全部の日で、日没はその日の正午より後・日の出より前で、
// 日没の時刻の太陽の高度は riseAltitude (誤差 0.01 度。二分法は 1 秒まで詰める)。
// 日没の 1 分前は上・1 分後は下、日の出はその逆。空を描く時刻は日没のちょうど 1 時間後。
func TestSpec_SunsetCrossing(t *testing.T) {
	c := Cities[0]
	alt := sunAltitude(c)
	for _, n := range tokyoNights(t) {
		noon := localNoon(c, n.Sunset)
		if !n.Sunset.After(noon) || !n.Sunrise.After(n.Sunset) || n.Sunrise.Sub(n.Sunset) > 24*time.Hour {
			t.Fatalf("日没 %v・日の出 %v", n.Sunset, n.Sunrise)
		}
		for _, ev := range []struct {
			at     time.Time
			rising bool
		}{{n.Sunset, false}, {n.Sunrise, true}} {
			if d := math.Abs(alt(ev.at) - riseAltitude); d > 0.01 {
				t.Fatalf("%v の高度のずれ %.4f", ev.at, d)
			}
			before, after := alt(ev.at.Add(-time.Minute)) > riseAltitude, alt(ev.at.Add(time.Minute)) > riseAltitude
			if before == ev.rising || after != ev.rising {
				t.Fatalf("%v: 横切る向きが違う", ev.at)
			}
		}
		if n.SkyAt.Sub(n.Sunset) != SkyDelay {
			t.Fatal(n.SkyAt, n.Sunset)
		}
	}
}

// TestSpec_ApproachVisible: 1 年の夜の 1 時間おきの全部の時刻で、ApproachAt が返すのは
// 月と惑星が両方地平線より上で、角距離が ApproachDeg 以内のときだけ。返さないときは、そういう惑星がない。
func TestSpec_ApproachVisible(t *testing.T) {
	c := Cities[0]
	found := 0
	for h := 0; h < 365*24; h += 1 {
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(h) * time.Hour)
		a := ApproachAt(Planets, c, at)
		m := MoonAt(c, at)
		closest := math.Inf(1)
		for _, p := range Planets {
			alt, az := HorizontalPlanet(p, c, at)
			if alt > 0 && m.Up() {
				closest = math.Min(closest, Separation(m.Alt, m.Az, alt, az))
			}
		}
		if a == nil {
			if closest <= ApproachDeg {
				t.Fatalf("%v: 角距離 %.2f の接近を見落とした", at, closest)
			}
			continue
		}
		found++
		alt, _ := HorizontalPlanet(a.Planet, c, at)
		if !m.Up() || alt <= 0 || a.Sep > ApproachDeg || a.Sep != closest {
			t.Fatalf("%v: %+v (月 %.1f・惑星 %.1f・いちばん近い %.2f)", at, a, m.Alt, alt, closest)
		}
	}
	if found == 0 {
		t.Fatal("1 年に 1 度も接近がない")
	}
}
