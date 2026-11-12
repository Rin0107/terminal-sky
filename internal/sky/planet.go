package sky

import (
	"math"
	"time"
)

// orbit は惑星の近似軌道要素 (J2000.0 の黄道と春分点)。どの要素も [J2000.0 の値, 1 ユリウス世紀あたりの変化]。
// a: 軌道長半径 [au]・e: 離心率・i: 軌道傾斜角 [度]・l: 平均黄経 [度]・peri: 近日点黄経 [度]・node: 昇交点黄経 [度]。
type orbit struct {
	a, e, i, l, peri, node [2]float64
}

// Planet は図に描く惑星。
type Planet struct {
	Name   string // 表示する名前 (凡例に出す)
	Symbol rune   // 図の記号 (全角 1 文字。東アジアの端末でも幅が 2 と決まっている漢字)
	orbit  orbit
	shine  shine
}

// shine は惑星の明るさの式の係数: 等級 = v0 + 5 log10(r・Δ) + c1・i + c2・i² + c3・i³。
// r は太陽から・Δ は地球からの距離 [au]、i は位相角 [度] (Meeus『天文アルゴリズム』41 章)。
// 土星は環の明るさを入れない (環が細く見える 2025〜2027 年は差が小さい)。
type shine struct {
	v0, c1, c2, c3 float64
}

// 出典: E. M. Standish, "Keplerian Elements for Approximate Positions of the Major Planets" (JPL Solar System Dynamics)
// の表 1 (1800〜2050 年に合わせた要素)。その範囲での誤差は角度で数分。2100 年までも 1 度より十分小さい。
var earthOrbit = orbit{
	a: [2]float64{1.00000261, 0.00000562}, e: [2]float64{0.01671123, -0.00004392}, i: [2]float64{-0.00001531, -0.01294668},
	l: [2]float64{100.46457166, 35999.37244981}, peri: [2]float64{102.93768193, 0.32327364}, node: [2]float64{0, 0},
}

// Planets は描く惑星 (肉眼で見える 5 つ)。並びは太陽に近い順で、凡例もこの順。
var Planets = []Planet{
	{"水星", '水', orbit{
		a: [2]float64{0.38709927, 0.00000037}, e: [2]float64{0.20563593, 0.00001906}, i: [2]float64{7.00497902, -0.00594749},
		l: [2]float64{252.25032350, 149472.67411175}, peri: [2]float64{77.45779628, 0.16047689}, node: [2]float64{48.33076593, -0.12534081},
	}, shine{-0.42, 0.0380, -0.000273, 0.000002}},
	{"金星", '金', orbit{
		a: [2]float64{0.72333566, 0.00000390}, e: [2]float64{0.00677672, -0.00004107}, i: [2]float64{3.39467605, -0.00078890},
		l: [2]float64{181.97909950, 58517.81538729}, peri: [2]float64{131.60246718, 0.00268329}, node: [2]float64{76.67984255, -0.27769418},
	}, shine{-4.40, 0.0009, 0.000239, -0.00000065}},
	{"火星", '火', orbit{
		a: [2]float64{1.52371034, 0.00001847}, e: [2]float64{0.09339410, 0.00007882}, i: [2]float64{1.84969142, -0.00813131},
		l: [2]float64{-4.55343205, 19140.30268499}, peri: [2]float64{-23.94362959, 0.44441088}, node: [2]float64{49.55953891, -0.29257343},
	}, shine{-1.52, 0.016, 0, 0}},
	{"木星", '木', orbit{
		a: [2]float64{5.20288700, -0.00011607}, e: [2]float64{0.04838624, -0.00013253}, i: [2]float64{1.30439695, -0.00183714},
		l: [2]float64{34.39644051, 3034.74612775}, peri: [2]float64{14.72847983, 0.21252668}, node: [2]float64{100.47390909, 0.20469106},
	}, shine{-9.40, 0.005, 0, 0}},
	{"土星", '土', orbit{
		a: [2]float64{9.53667594, -0.00125060}, e: [2]float64{0.05386179, -0.00050991}, i: [2]float64{2.48599187, 0.00193609},
		l: [2]float64{49.95424423, 1222.49362201}, peri: [2]float64{92.59887831, -0.41897216}, node: [2]float64{113.66242448, -0.28867794},
	}, shine{-8.88, 0, 0, 0}},
}

// obliquityJ2000 は J2000.0 の黄道傾斜角 [度]。
const obliquityJ2000 = 23.43928

// elementAt は要素 [値, 変化] の、t ユリウス世紀後の値。
func elementAt(el [2]float64, t float64) float64 { return el[0] + el[1]*t }

// solveKepler はケプラーの方程式 E - e・sin E = M を、平均近点角 m [ラジアン] から離心近点角 E [ラジアン] について解く。
// 離心率 0.25 以下ではニュートン法 8 回で 1e-12 より小さくなる (TestSpec_Kepler)。
func solveKepler(m, e float64) float64 {
	ea := m + e*math.Sin(m)
	for range 8 {
		ea -= (ea - e*math.Sin(ea) - m) / (1 - e*math.Cos(ea))
	}
	return ea
}

// heliocentric は jd の日の太陽からの位置 [au] (J2000.0 の黄道座標)。
func (o orbit) heliocentric(jd float64) (x, y, z float64) {
	t := (jd - j2000) / 36525
	a, e := elementAt(o.a, t), elementAt(o.e, t)
	inc, peri, node := elementAt(o.i, t)*deg, elementAt(o.peri, t)*deg, elementAt(o.node, t)*deg
	m := norm360(elementAt(o.l, t)-elementAt(o.peri, t)) * deg
	ea := solveKepler(m, e)
	xp, yp := a*(math.Cos(ea)-e), a*math.Sqrt(1-e*e)*math.Sin(ea)
	w := peri - node
	cw, sw, cn, sn, ci, si := math.Cos(w), math.Sin(w), math.Cos(node), math.Sin(node), math.Cos(inc), math.Sin(inc)
	x = (cw*cn-sw*sn*ci)*xp + (-sw*cn-cw*sn*ci)*yp
	y = (cw*sn+sw*cn*ci)*xp + (-sw*sn+cw*cn*ci)*yp
	z = sw*si*xp + cw*si*yp
	return x, y, z
}

// eclipticToEquatorial は黄道座標を、黄道傾斜角 eps [度] で赤道座標に回す。
func eclipticToEquatorial(x, y, z, eps float64) (xe, ye, ze float64) {
	ce, se := math.Cos(eps*deg), math.Sin(eps*deg)
	return x, y*ce - z*se, y*se + z*ce
}

// toRADec は赤道座標の向きを赤経・赤緯 [度] に直す。
func toRADec(x, y, z float64) (ra, dec float64) {
	return norm360(math.Atan2(y, x) / deg), math.Atan2(z, math.Hypot(x, y)) / deg
}

// Equatorial は jd の日の、地球の中心から見た惑星の赤経・赤緯 [度] (J2000.0 の赤道と春分点。光の時間は入れない)。
func (p Planet) Equatorial(jd float64) (ra, dec float64) {
	px, py, pz := p.orbit.heliocentric(jd)
	ex, ey, ez := earthOrbit.heliocentric(jd)
	return toRADec(eclipticToEquatorial(px-ex, py-ey, pz-ez, obliquityJ2000))
}

// Magnitude は jd の日の、地球から見た惑星の等級 (小さいほど明るい)。
func (p Planet) Magnitude(jd float64) float64 {
	px, py, pz := p.orbit.heliocentric(jd)
	ex, ey, ez := earthOrbit.heliocentric(jd)
	r := math.Sqrt(px*px + py*py + pz*pz)
	sunEarth := math.Sqrt(ex*ex + ey*ey + ez*ez)
	dx, dy, dz := px-ex, py-ey, pz-ez
	delta := math.Sqrt(dx*dx + dy*dy + dz*dz)
	cosI := (r*r + delta*delta - sunEarth*sunEarth) / (2 * r * delta)
	i := math.Acos(math.Max(-1, math.Min(1, cosI))) / deg
	s := p.shine
	return s.v0 + 5*math.Log10(r*delta) + s.c1*i + s.c2*i*i + s.c3*i*i*i
}

// PlanetSeen は惑星 1 つの、都市から見た高度・方位 [度]。
type PlanetSeen struct {
	Planet Planet
	Alt    float64
	Az     float64
}

// HorizontalPlanet は惑星の高度・方位 [度] を返す。
func HorizontalPlanet(p Planet, c City, t time.Time) (alt, az float64) {
	jd := JulianDay(t)
	ra0, dec0 := p.Equatorial(jd)
	ra, dec := Precess(ra0, dec0, jd)
	return horizontalOfDate(ra, dec, c, jd)
}

// VisiblePlanets は地平線より上の惑星を、planets の順に返す。
func VisiblePlanets(planets []Planet, c City, t time.Time) []PlanetSeen {
	var out []PlanetSeen
	for _, p := range planets {
		if alt, az := HorizontalPlanet(p, c, t); alt > 0 {
			out = append(out, PlanetSeen{Planet: p, Alt: alt, Az: az})
		}
	}
	return out
}
