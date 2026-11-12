package sky

import (
	"errors"
	"fmt"
	"math"
	"time"
)

// riseAltitude は日の出入り・月の出入りの、天体の中心の高度 [度]。
// 大気の屈折 (34') と見かけの半径 (約 16') の分だけ地平線より下で、上の縁が地平線に触れる (国立天文台の暦と同じ定義)。
// 月の高度は MoonAt で視差を入れてあるので、太陽と同じ値でよい。
const riseAltitude = -0.833

// 今夜の見どころの決まり。
const (
	// SkyDelay は日没から、空を描く時刻までの時間。空が暗くなり、明るい星が見えてくるころ。
	SkyDelay = time.Hour
	// PeakAltitude は「見える」と数える惑星の高さ [度]。低すぎると建物や霞に隠れる。
	PeakAltitude = 10.0
	// ApproachDeg は「接近」と数える月と惑星の角距離 [度]。
	ApproachDeg = 5.0
	// scanStep は夜を見ていく刻み。太陽と月の高度は 10 分で 3 度も変わらないので、出入りを見落とさない。
	scanStep = 10 * time.Minute
)

// ErrNoNight は、その日に日が沈まない・昇らない (白夜・極夜) ときの誤り。
var ErrNoNight = errors.New("この日は日が沈まないか昇らないので、今夜の空を決められない")

// SunAt は都市 c から時刻 t に見た太陽の高度・方位 [度]。
func SunAt(c City, t time.Time) (alt, az float64) {
	jd := JulianDay(t)
	lon := SunLongitude(jd)
	eps := obliquityJ2000 - 0.0130042*(jd-j2000)/36525
	ra, dec := toRADec(eclipticToEquatorial(cosDeg(lon), sinDeg(lon), 0, eps))
	return horizontalOfDate(ra, dec, c, jd)
}

// altitudeFunc は時刻から天体の高度 [度] を返す関数。
type altitudeFunc func(time.Time) float64

func sunAltitude(c City) altitudeFunc {
	return func(t time.Time) float64 { alt, _ := SunAt(c, t); return alt }
}

func moonAltitude(c City) altitudeFunc {
	return func(t time.Time) float64 { return MoonAt(c, t).Alt }
}

// crossing は、a では高度が h より上か下で、b ではその逆のとき、間で高度が h を横切る時刻を 1 秒以内まで二分法で求める。
func crossing(alt altitudeFunc, h float64, a, b time.Time) time.Time {
	above := alt(a) > h
	for b.Sub(a) > time.Second {
		m := a.Add(b.Sub(a) / 2)
		if (alt(m) > h) == above {
			a = m
		} else {
			b = m
		}
	}
	return b
}

// nextPassage は from から to までを scanStep 刻みで見て、高度が h を最初に横切る時刻を返す。
// rising なら下から上 (出)、そうでなければ上から下 (入り)。なければ false。
func nextPassage(alt altitudeFunc, h float64, from, to time.Time, rising bool) (time.Time, bool) {
	prev := alt(from) > h
	for t := from; t.Before(to); {
		n := t.Add(scanStep)
		if n.After(to) {
			n = to
		}
		cur := alt(n) > h
		if cur != prev && cur == rising {
			return crossing(alt, h, t, n), true
		}
		prev, t = cur, n
	}
	return time.Time{}, false
}

// Sunset は、都市 c の暦の日 day (c の時刻の帯で数える) の日没の時刻。正午から 24 時間のうちに沈まなければ false。
func Sunset(c City, day time.Time) (time.Time, bool) {
	noon := localNoon(c, day)
	return nextPassage(sunAltitude(c), riseAltitude, noon, noon.Add(24*time.Hour), false)
}

// localNoon は都市 c の時刻の帯で、day と同じ暦の日の 12 時。
func localNoon(c City, day time.Time) time.Time {
	y, m, d := day.In(c.Zone).Date()
	return time.Date(y, m, d, 12, 0, 0, 0, c.Zone)
}

// Moment は起きるかどうか分からない出来事の時刻 (月の出・月の入り)。
type Moment struct {
	At time.Time
	OK bool // この夜に起きるか
}

// PlanetPeak は、今夜いちばん明るく見える惑星と、いちばん高く昇る時刻・方位・高さ。
type PlanetPeak struct {
	Planet Planet
	Mag    float64 // 等級
	At     time.Time
	Alt    float64 // 高さ [度]
	Az     float64 // 方位 [度]
}

// Approach は月と惑星の接近 (角距離 ApproachDeg 以内)。
type Approach struct {
	Planet Planet
	At     time.Time
	Sep    float64 // 月との角距離 [度]
	Alt    float64 // 月の高さ [度]
	Az     float64 // 月の方位 [度]
}

// Night は、ある日の夜 (日没から翌朝の日の出まで) の見どころ。
type Night struct {
	City      City
	Sunset    time.Time
	Sunrise   time.Time
	SkyAt     time.Time // 空を描く時刻 (日没の SkyDelay 後)
	MoonAge   float64   // SkyAt の月齢 [日]
	Moonrise  Moment    // 正午から翌日の正午までの最初の月の出
	Moonset   Moment    // 同じく、最初の月の入り
	Brightest *PlanetPeak
	Approach  *Approach
}

// Tonight は都市 c の暦の日 day の夜の見どころを計算する。
// 惑星と接近は、空を描く時刻から日の出の 1 時間前まで (暗い間) を scanStep 刻みで見る。
func Tonight(c City, day time.Time) (Night, error) {
	sunset, ok := Sunset(c, day)
	if !ok {
		return Night{}, ErrNoNight
	}
	sunrise, ok := nextPassage(sunAltitude(c), riseAltitude, sunset, sunset.Add(24*time.Hour), true)
	if !ok {
		return Night{}, ErrNoNight
	}
	n := Night{City: c, Sunset: sunset, Sunrise: sunrise, SkyAt: sunset.Add(SkyDelay)}
	n.MoonAge = MoonAge(JulianDay(n.SkyAt))
	noon := localNoon(c, day)
	n.Moonrise.At, n.Moonrise.OK = nextPassage(moonAltitude(c), riseAltitude, noon, noon.Add(24*time.Hour), true)
	n.Moonset.At, n.Moonset.OK = nextPassage(moonAltitude(c), riseAltitude, noon, noon.Add(24*time.Hour), false)
	samples := darkSamples(n.SkyAt, sunrise.Add(-SkyDelay))
	n.Brightest = brightest(Planets, c, samples)
	n.Approach = closestApproach(Planets, c, samples)
	return n, nil
}

// darkSamples は from から to まで scanStep 刻みの時刻 (from は必ず含む。to が from より前でも from だけ)。
func darkSamples(from, to time.Time) []time.Time {
	out := []time.Time{from}
	for t := from.Add(scanStep); !t.After(to); t = t.Add(scanStep) {
		out = append(out, t)
	}
	return out
}

// brightest は、samples のどれかで高さ PeakAltitude 以上に昇る惑星のうち、いちばん明るい (等級が小さい) ものと、
// それがいちばん高い時刻を返す。そういう惑星がなければ nil。等級は最初の時刻で比べる。
func brightest(planets []Planet, c City, samples []time.Time) *PlanetPeak {
	var best *PlanetPeak
	jd := JulianDay(samples[0])
	for _, p := range planets {
		peak := PlanetPeak{Planet: p, Mag: p.Magnitude(jd), Alt: math.Inf(-1)}
		for _, t := range samples {
			if alt, az := HorizontalPlanet(p, c, t); alt > peak.Alt {
				peak.At, peak.Alt, peak.Az = t, alt, az
			}
		}
		if highEnough(peak.Alt) && (best == nil || peak.Mag < best.Mag) {
			best = &peak
		}
	}
	return best
}

// highEnough は、惑星の高さ [度] が「見える」と数える PeakAltitude 以上か (ちょうど 10 度は見える)。
func highEnough(alt float64) bool { return alt >= PeakAltitude }

// approaching は、月と惑星の角距離 [度] が「接近」と数える ApproachDeg 以内か (ちょうど 5 度は接近)。
func approaching(sep float64) bool { return sep <= ApproachDeg }

// closestApproach は、samples の中で月と惑星がいちばん近づいた接近 (角距離 ApproachDeg 以内) を返す。なければ nil。
func closestApproach(planets []Planet, c City, samples []time.Time) *Approach {
	var best *Approach
	for _, t := range samples {
		if a := ApproachAt(planets, c, t); a != nil && (best == nil || a.Sep < best.Sep) {
			best = a
		}
	}
	return best
}

// ApproachAt は時刻 t に、地平線より上で月から角距離 ApproachDeg 以内にある惑星のうち、いちばん近いものを返す。
// 月か惑星が地平線より下なら数えない (見えないので)。なければ nil。
func ApproachAt(planets []Planet, c City, t time.Time) *Approach {
	m := MoonAt(c, t)
	if !m.Up() {
		return nil
	}
	var best *Approach
	for _, p := range planets {
		alt, az := HorizontalPlanet(p, c, t)
		if alt <= 0 {
			continue
		}
		if sep := Separation(m.Alt, m.Az, alt, az); approaching(sep) && (best == nil || sep < best.Sep) {
			best = &Approach{Planet: p, At: t, Sep: sep, Alt: m.Alt, Az: m.Az}
		}
	}
	return best
}

// Separation は、高度・方位 [度] で表した 2 つの向きの角距離 [度、0〜180]。
func Separation(alt1, az1, alt2, az2 float64) float64 {
	c := sinDeg(alt1)*sinDeg(alt2) + cosDeg(alt1)*cosDeg(alt2)*cosDeg(az1-az2)
	return math.Acos(math.Max(-1, math.Min(1, c))) / deg
}

// compassPoints は北から東回りの 16 方位。
var compassPoints = [16]string{
	"北", "北北東", "北東", "東北東", "東", "東南東", "南東", "南南東",
	"南", "南南西", "南西", "西南西", "西", "西北西", "北西", "北北西",
}

// compassIndex は方位 [度] を 16 方位の番号 (0 = 北、4 = 東、8 = 南、12 = 西) にする。
// 番号 k の方位は k・22.5 度を中心に ±11.25 度。範囲の外の方位も 360 度で折り返す。仕様: spec/sky.smt2 の 9。
func compassIndex(az float64) int {
	return int(math.Floor(norm360(az)/22.5+0.5)) % 16
}

// Compass16 は方位 [度] を 16 方位の日本語 (例: 南東) にする。
func Compass16(az float64) string { return compassPoints[compassIndex(az)] }

// phaseNames は月の 8 つの相 (moonPhase の番号) の呼び名。
var phaseNames = [8]string{"新月", "三日月", "上弦", "十三夜", "満月", "寝待月", "下弦", "有明月"}

// PhaseName は月齢 [日] の相の呼び名。
func PhaseName(age float64) string { return phaseNames[moonPhase(age)] }

// clock は時刻を都市の時刻の帯の時・分にする。
func clock(c City, t time.Time) string { return t.In(c.Zone).Format("15:04") }

// moment は月の出・月の入りの時刻か、この夜にないなら「なし」。
func moment(c City, name string, m Moment) string {
	if !m.OK {
		return name + "なし"
	}
	return name + " " + clock(c, m.At)
}

// fitFirst は、長い順に並べた言い方 forms のうち、幅 width に入る最初のものを返す。
// どれも入らなければ、いちばん短いものを幅で切る。
func fitFirst(forms []string, width int) string {
	for _, f := range forms {
		if Width(f) <= width {
			return f
		}
	}
	return Truncate(forms[len(forms)-1], width)
}

// planetForms は一番明るい惑星の行の言い方 (長い順)。どれも方角と高さを残す (空と見比べて探すため)。
func planetForms(c City, b *PlanetPeak) []string {
	if b == nil {
		return []string{fmt.Sprintf("今夜は高さ %.0f 度より上に昇る惑星がない", PeakAltitude), "高く昇る惑星なし"}
	}
	name, at, dir := b.Planet.Name, clock(c, b.At), Compass16(b.Az)
	return []string{
		fmt.Sprintf("一番明るい惑星は%s (%.1f 等): %s ごろ%sの空、高さ %.0f 度", name, b.Mag, at, dir, b.Alt),
		fmt.Sprintf("%s %.1f 等 %s %s %.0f 度", name, b.Mag, at, dir, b.Alt),
		fmt.Sprintf("%s %s %s %.0f 度", name, at, dir, b.Alt),
		fmt.Sprintf("%s %s %.0f 度", name, dir, b.Alt),
	}
}

// moonForms は月齢と月の出入りの行の言い方 (長い順)。
func moonForms(n Night) []string {
	c := n.City
	rise, set := moment(c, "出", n.Moonrise), moment(c, "入", n.Moonset) // 狭い端末での言い方 (例: "出 04:55"・"入なし")
	return []string{
		fmt.Sprintf("月齢 %.1f (%s)  %s  %s", n.MoonAge, PhaseName(n.MoonAge), moment(c, "月の出", n.Moonrise), moment(c, "月の入り", n.Moonset)),
		fmt.Sprintf("月齢 %.1f %s %s", n.MoonAge, rise, set),
		rise + " " + set,
	}
}

// approachForms は接近の行の言い方 (長い順)。どれも方角を残す。
func approachForms(c City, a *Approach) []string {
	name, at, dir := a.Planet.Name, clock(c, a.At), Compass16(a.Az)
	return []string{
		fmt.Sprintf("月と%sが接近: %s ごろ%sの空、角距離 %.1f 度", name, at, dir, a.Sep),
		fmt.Sprintf("月と%s 接近 %s %s %.1f 度", name, at, dir, a.Sep),
		fmt.Sprintf("月と%s 接近 %s", name, dir),
	}
}

// Highlights は今夜の見どころの文 (2〜3 行、どれも幅 width 以下): 一番明るい惑星・月齢と月の出入り・月と惑星の接近 (あれば)。
// 方角は 16 方位の日本語。幅に入らなければ、方角を残したまま段階的に短い言い方にする (幅 20 でも方角は消えない)。
func Highlights(n Night, width int) []string {
	lines := []string{fitFirst(planetForms(n.City, n.Brightest), width), fitFirst(moonForms(n), width)}
	if a := n.Approach; a != nil {
		lines = append(lines, fitFirst(approachForms(n.City, a), width))
	}
	return lines
}
