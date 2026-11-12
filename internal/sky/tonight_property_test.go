package sky

// 今夜の見どころの性質を、ランダムな入力で確かめる (PBT と fuzz)。

import (
	"math"
	"testing"
	"testing/quick"
	"time"
)

// 角距離は 0〜180 度で、向きを入れ替えても同じ。同じ向きなら 0。
func TestSeparationProperties(t *testing.T) {
	prop := func(a1, z1, a2, z2 float64) bool {
		alt1, az1 := math.Mod(a1, 90), math.Mod(z1, 360)
		alt2, az2 := math.Mod(a2, 90), math.Mod(z2, 360)
		s := Separation(alt1, az1, alt2, az2)
		return s >= 0 && s <= 180 && math.Abs(s-Separation(alt2, az2, alt1, az1)) < 1e-9 &&
			Separation(alt1, az1, alt1, az1) < 1e-4
	}
	if err := quick.Check(prop, &quick.Config{MaxCount: 5000}); err != nil {
		t.Fatal(err)
	}
	// 境界値: 天頂と地平線は 90 度、反対の方位の地平線どうしは 180 度
	if d := Separation(90, 0, 0, 123); math.Abs(d-90) > 1e-9 {
		t.Fatal(d)
	}
	if d := Separation(0, 0, 0, 180); math.Abs(d-180) > 1e-9 {
		t.Fatal(d)
	}
}

// どの日 (1900〜2100 年) でも、東京の夜の見どころは止まらずに出て、1〜3 行で、どの行も幅 (20〜400) に入る。
func TestTonightAnyDay(t *testing.T) {
	c := Cities[0]
	prop := func(days uint32) bool {
		day := time.Date(1900, 1, 1, 0, 0, 0, 0, c.Zone).AddDate(0, 0, int(days%(201*365)))
		w := 20 + int(days%381)
		n, err := Tonight(c, day)
		if err != nil {
			return false
		}
		lines := Highlights(n, w)
		for _, l := range lines {
			if Width(l) > w {
				return false
			}
		}
		return len(lines) >= 1 && len(lines) <= 3 && n.Sunrise.After(n.Sunset)
	}
	if err := quick.Check(prop, &quick.Config{MaxCount: 60}); err != nil {
		t.Fatal(err)
	}
}

func FuzzCompass16(f *testing.F) {
	for _, az := range []float64{0, 11.25, 348.75, 359.999, -0.001, 1e9, -1e9} {
		f.Add(az)
	}
	f.Fuzz(func(t *testing.T, az float64) {
		if math.IsNaN(az) || math.IsInf(az, 0) {
			return
		}
		k := compassIndex(az)
		if k < 0 || k > 15 || Compass16(az) == "" {
			t.Fatalf("%v: %d", az, k)
		}
	})
}
