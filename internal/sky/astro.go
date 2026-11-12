package sky

import (
	"math"
	"sort"
	"strings"
	"time"
)

// City は内蔵の観測地。利用者の現在地は受け取らず、ここから名前で選ぶだけ。
type City struct {
	Key    string         // --city で指定する名前
	Label  string         // 表示する名前
	LatDeg float64        // 緯度 [度] (北が正)
	LonDeg float64        // 経度 [度] (東が正)
	Zone   *time.Location // 表示に使う時刻の帯
}

var jst = time.FixedZone("JST", 9*60*60)

// Cities は内蔵の都市。まずは東京だけ。
var Cities = []City{
	{Key: "tokyo", Label: "東京", LatDeg: 35.6895, LonDeg: 139.6917, Zone: jst},
}

// FindCity は名前 (大文字小文字は区別しない) で都市を探す。
func FindCity(key string) (City, bool) {
	k := strings.ToLower(strings.TrimSpace(key))
	for _, c := range Cities {
		if c.Key == k {
			return c, true
		}
	}
	return City{}, false
}

// CityKeys は選べる都市の名前の一覧。
func CityKeys() []string {
	keys := make([]string, 0, len(Cities))
	for _, c := range Cities {
		keys = append(keys, c.Key)
	}
	return keys
}

const (
	deg = math.Pi / 180
	// j2000 は J2000.0 のユリウス日。
	j2000 = 2451545.0
)

// JulianDay は時刻をユリウス日 (UT) に直す。
// 秒とナノ秒を分けて足す (UnixNano は 1678 年より前・2262 年より後であふれるため)。
func JulianDay(t time.Time) float64 {
	return float64(t.Unix())/86400 + float64(t.Nanosecond())/86400e9 + 2440587.5
}

// norm360 は角度を [0, 360) に入れる。
func norm360(a float64) float64 {
	a = math.Mod(a, 360)
	if a < 0 {
		a += 360
	}
	if a == 360 { // ごく小さい負の数に 360 を足すと、丸めで 360 ちょうどになる
		return 0
	}
	return a
}

// GMST はグリニッジ平均恒星時 [度] (Meeus『天文アルゴリズム』12.4)。
func GMST(jd float64) float64 {
	d := jd - j2000
	t := d / 36525
	return norm360(280.46061837 + 360.98564736629*d + 0.000387933*t*t - t*t*t/38710000)
}

// Precess は J2000.0 の赤経・赤緯 [度] を、jd の日の平均位置へ歳差で動かす (Meeus 21.2〜21.4)。
func Precess(raDeg, decDeg, jd float64) (float64, float64) {
	t := (jd - j2000) / 36525
	zeta := (2306.2181*t + 0.30188*t*t + 0.017998*t*t*t) / 3600 * deg
	z := (2306.2181*t + 1.09468*t*t + 0.018203*t*t*t) / 3600 * deg
	theta := (2004.3109*t - 0.42665*t*t - 0.041833*t*t*t) / 3600 * deg
	a0, d0 := raDeg*deg, decDeg*deg
	a := math.Cos(d0) * math.Sin(a0+zeta)
	b := math.Cos(theta)*math.Cos(d0)*math.Cos(a0+zeta) - math.Sin(theta)*math.Sin(d0)
	c := math.Sin(theta)*math.Cos(d0)*math.Cos(a0+zeta) + math.Cos(theta)*math.Sin(d0)
	ra := norm360((math.Atan2(a, b) + z) / deg)
	dec := math.Asin(math.Max(-1, math.Min(1, c))) / deg
	return ra, dec
}

// Horizontal は星の高度 [度, -90〜90] と方位 [度, 北から東回りに 0〜360) を返す。
func Horizontal(s Star, c City, t time.Time) (alt, az float64) {
	jd := JulianDay(t)
	ra, dec := Precess(s.RA*15, s.Dec, jd)
	return horizontalOfDate(ra, dec, c, jd)
}

// horizontalOfDate は jd の日の赤経・赤緯 [度] を、都市 c から見た高度・方位 [度] に直す。
func horizontalOfDate(ra, dec float64, c City, jd float64) (alt, az float64) {
	lst := GMST(jd) + c.LonDeg
	h := (lst - ra) * deg
	phi, d := c.LatDeg*deg, dec*deg
	up := math.Sin(phi)*math.Sin(d) + math.Cos(phi)*math.Cos(d)*math.Cos(h)
	east := -math.Cos(d) * math.Sin(h)
	north := math.Sin(d)*math.Cos(phi) - math.Cos(d)*math.Cos(h)*math.Sin(phi)
	alt = math.Asin(math.Max(-1, math.Min(1, up))) / deg
	az = norm360(math.Atan2(east, north) / deg)
	return alt, az
}

// Seen は地平線より上に見えている星 1 つ。
type Seen struct {
	Star Star
	Alt  float64
	Az   float64
}

// Visible は地平線より上の星を、明るい順 (同じ等級なら目録の順) に返す。
func Visible(stars []Star, c City, t time.Time) []Seen {
	var out []Seen
	for _, s := range stars {
		alt, az := Horizontal(s, c, t)
		if alt > 0 {
			out = append(out, Seen{Star: s, Alt: alt, Az: az})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Star.Mag < out[j].Star.Mag })
	return out
}
