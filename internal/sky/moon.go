package sky

import (
	"math"
	"time"
)

// SynodicMonth は朔望月 (新月から次の新月まで) の平均の長さ [日]。
const SynodicMonth = 29.530588853

// earthRadiusKm は地球の赤道半径 [km] (月の視差に使う)。
const earthRadiusKm = 6378.14

// sinDeg は度で受け取る sin。
func sinDeg(a float64) float64 { return math.Sin(a * deg) }

// cosDeg は度で受け取る cos。
func cosDeg(a float64) float64 { return math.Cos(a * deg) }

// SunLongitude は jd の日の太陽の視黄経 [度、その日の黄道と春分点] (Meeus『天文アルゴリズム』25 章の簡易式、誤差 0.01 度ほど)。
func SunLongitude(jd float64) float64 {
	t := (jd - j2000) / 36525
	l0 := 280.46646 + 36000.76983*t
	m := 357.52911 + 35999.05029*t
	c := (1.914602-0.004817*t)*sinDeg(m) + 0.019993*sinDeg(2*m) + 0.000289*sinDeg(3*m)
	return norm360(l0 + c)
}

// MoonEcliptic は jd の日の月の黄経・黄緯 [度、その日の黄道と春分点] と、地球の中心からの距離 [km]。
// Meeus『天文アルゴリズム』47 章の級数の大きい項だけ (簡易式。誤差は黄経で 0.3 度ほど)。
func MoonEcliptic(jd float64) (lon, lat, distKm float64) {
	t := (jd - j2000) / 36525
	lp := 218.3164477 + 481267.88123421*t // 月の平均黄経
	d := 297.8501921 + 445267.1114034*t   // 月と太陽の平均離角
	m := 357.5291092 + 35999.0502909*t    // 太陽の平均近点角
	mp := 134.9633964 + 477198.8675055*t  // 月の平均近点角
	f := 93.2720950 + 483202.0175233*t    // 月の昇交点からの平均角距離
	lon = lp + 6.288774*sinDeg(mp) + 1.274027*sinDeg(2*d-mp) + 0.658314*sinDeg(2*d) +
		0.213618*sinDeg(2*mp) - 0.185116*sinDeg(m) - 0.114332*sinDeg(2*f) +
		0.058793*sinDeg(2*d-2*mp) + 0.057066*sinDeg(2*d-m-mp) + 0.053322*sinDeg(2*d+mp) +
		0.045758*sinDeg(2*d-m) - 0.040923*sinDeg(m-mp) - 0.034720*sinDeg(d) - 0.030383*sinDeg(m+mp)
	lat = 5.128122*sinDeg(f) + 0.280602*sinDeg(mp+f) + 0.277693*sinDeg(mp-f) + 0.173237*sinDeg(2*d-f) +
		0.055413*sinDeg(2*d-mp+f) + 0.046271*sinDeg(2*d-mp-f) + 0.032573*sinDeg(2*d+f)
	distKm = 385000.56 - 20905.355*cosDeg(mp) - 3699.111*cosDeg(2*d-mp) - 2955.968*cosDeg(2*d) - 569.925*cosDeg(2*mp)
	return norm360(lon), lat, distKm
}

// ageFromElongation は、太陽から東へ測った月の離角 [度] を月齢 [日、0 以上 SynodicMonth 未満] に直す。
// 仕様: spec/sky.smt2 の 7。
func ageFromElongation(elong float64) float64 {
	return norm360(elong) / 360 * SynodicMonth
}

// MoonAge は jd の日の月齢 [日]。月が太陽から東へ離れた角度を、平均の朔望月に比べて日に直す
// (新月で 0、上弦で約 7.4、満月で約 14.8)。
func MoonAge(jd float64) float64 {
	lon, _, _ := MoonEcliptic(jd)
	return ageFromElongation(lon - SunLongitude(jd))
}

// moonSymbols は月齢を 8 つに分けた記号。北半球で見た明るい側の形 (右が満ちる側):
// 新月 o・三日月 )・上弦 D・十三夜 O・満月 O・寝待月 O・下弦 C・有明月 (。
// 星 (● * +)・星座線 (- | / \)・地平線 (.) と重ならない ASCII だけ。
var moonSymbols = [8]rune{'o', ')', 'D', 'O', 'O', 'O', 'C', '('}

// moonPhase は月齢を 8 つの相 (0 = 新月、4 = 満月) に分ける。朔望月の外の月齢も、周期で折り返して 0〜7 に入れる。
// 仕様: spec/sky.smt2 の 7。
func moonPhase(age float64) int {
	p := int(math.Floor(age/SynodicMonth*8+0.5)) % 8
	return (p + 8) % 8
}

// MoonSymbol は月齢 [日] に応じた月の記号。
func MoonSymbol(age float64) rune { return moonSymbols[moonPhase(age)] }

// MoonSeen は都市から見た月。高度は地平線より下のこともある (Up で確かめる)。
type MoonSeen struct {
	Alt float64 // 高度 [度] (視差を入れた、都市から見た高度)
	Az  float64 // 方位 [度]
	Age float64 // 月齢 [日]
}

// Up は月が地平線より上にあるか。
func (m MoonSeen) Up() bool { return m.Alt > 0 }

// MoonAt は都市 c から時刻 t に見た月の高度・方位・月齢。
// 月は近いので、地球の中心から見た高度を視差 (最大 1 度ほど) の分だけ下げる。
func MoonAt(c City, t time.Time) MoonSeen {
	jd := JulianDay(t)
	lon, lat, dist := MoonEcliptic(jd)
	eps := obliquityJ2000 - 0.0130042*(jd-j2000)/36525 // その日の黄道傾斜角
	x, y, z := cosDeg(lat)*cosDeg(lon), cosDeg(lat)*sinDeg(lon), sinDeg(lat)
	ra, dec := toRADec(eclipticToEquatorial(x, y, z, eps))
	alt, az := horizontalOfDate(ra, dec, c, jd)
	parallax := math.Asin(earthRadiusKm/dist) / deg
	return MoonSeen{Alt: alt - parallax*cosDeg(alt), Az: az, Age: MoonAge(jd)}
}
