package sky

import (
	"math"
	"strings"
)

// Facing は図の向き。AllSky (既定) は空を真上から見た全天の図、ほかはその方角を向いて
// 地平線 (図の下の縁) から天頂 (上の縁) までを見た図 (左右に 90 度ずつ、合わせて 180 度)。
type Facing int

const (
	AllSky Facing = iota
	North
	East
	South
	West
)

var (
	facingKeys   = [...]string{AllSky: "", North: "north", East: "east", South: "south", West: "west"}
	facingLabels = [...]string{AllSky: "", North: "北", East: "東", South: "南", West: "西"}
	// compass8 は 8 方位の略号 (北から時計回り)。方角を向いた図の地平線に書く (全天の図の N E S W と同じ英字)。
	compass8 = [8]string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
)

// FacingKeys は --facing に書ける値 (北から時計回り)。
func FacingKeys() []string { return append([]string{}, facingKeys[North:]...) }

// ParseFacing は --facing の値 (north・east・south・west。大文字小文字は区別しない) を読む。
func ParseFacing(s string) (Facing, bool) {
	k := strings.ToLower(strings.TrimSpace(s))
	for f := North; f <= West; f++ {
		if facingKeys[f] == k {
			return f, true
		}
	}
	return AllSky, false
}

// Label は向きの日本語 (北・東・南・西)。全天は空。
func (f Facing) Label() string { return facingLabels[f] }

// az は向きの方位 [度] (北 0・東 90・南 180・西 270)。
func (f Facing) az() float64 { return float64(f-North) * 90 }

// Shows は高度 alt・方位 az のものが、この向きの図に入るか。
// 全天の図は地平線より上の全部、方角を向いた図はそのうち向きから左右 90 度以内のもの。
func (f Facing) Shows(alt, az float64) bool {
	return alt > 0 && (f == AllSky || math.Abs(relAz(az, f.az())) <= 90)
}

// relAz は方位 az を、向き center から見た角 [度, -180 以上 180 未満] にする (右 = 時計回りが正)。
// 仕様: spec/sky.smt2 の 12。
func relAz(az, center float64) float64 { return norm360(az-center+180) - 180 }

// facingView は方角を向いた図。行は高度 (上の縁が天頂・行 bottom が高度 0)、
// 列は向きからの角 rel に cos(高度) を掛けたもの (サンソン図法。地平線では左の縁 -90 度・真ん中 0 度・右の縁 +90 度、
// 高くなるほど真ん中に寄り、天頂は 1 点)。いちばん下の行 (bottom + 1) は地平線と方角。
// 方位をそのまま列にすると、天頂の近くで隣り合う星が図の端と端に離れ、星座線が横に長く伸びてしまうので、高さで縮める。
// 端末の文字は縦長 (1 行 ≒ 2 列) なので、80 × 22 なら地平線の 180 度が 79 列・90 度が 21 行で、縦横の見た目がそろう。
type facingView struct {
	center float64 // 向きの方位 [度]
	cx, hx int     // 真ん中の列と、真ん中から 90 度までの列の数
	bottom int     // 高度 0 の行
}

func newFacingView(f Facing, w, h int) facingView {
	return facingView{center: f.az(), cx: (w - 1) / 2, hx: (w - 1) / 2, bottom: h - 2}
}

// pos は高度 alt・方位 az を升 (行, 列) にする。仕様: spec/sky.smt2 の 11 (図に入るものは格子の中)。
func (v facingView) pos(alt, az float64) (row, col int) {
	row = int(math.Round((90 - alt) / 90 * float64(v.bottom)))
	col = v.cx + int(math.Round(relAz(az, v.center)*math.Cos(alt*deg)/90*float64(v.hx)))
	return row, col
}

// halfWidth は行 row で、向きから左右 90 度までが真ん中から何列か (高度 a の行では hx・cos(a))。
func (v facingView) halfWidth(row int) int {
	a := 90 - float64(row)/float64(v.bottom)*90
	return int(math.Round(float64(v.hx) * math.Cos(a*deg)))
}

func (v facingView) shows(alt, az float64) bool {
	return alt > 0 && math.Abs(relAz(az, v.center)) <= 90
}

// inside は線の点を置いてよい升か (地平線の行より上で、左右 90 度の中)。背中側へ伸びる線は 90 度の縁で切る。
func (v facingView) inside(row, col int) bool {
	return 0 <= row && row <= v.bottom && abs(col-v.cx) <= v.halfWidth(row)
}

// crosses は線が背中側 (向きの真後ろ) をまたぐか。またぐ線を結ぶと図の端から端へ横切ってしまう。
// 仕様: spec/sky.smt2 の 12。
func (v facingView) crosses(a, b Seen) bool {
	return math.Abs(relAz(a.Az, v.center)-relAz(b.Az, v.center)) > 180
}

// frame は地平線 (いちばん下の行の点) と、45 度ごとの方角 (左の縁・真ん中・右の縁と、その間) を描く。
// 地平線の行は名前に譲らない (全天の図と違い、まっすぐな 1 行なので、欠けると方角の目盛りが読みにくい)。
func (v facingView) frame(g *grid) {
	row := v.bottom + 1
	for c := v.cx - v.hx; c <= v.cx+v.hx; c++ {
		g.cells[row][c] = cell{r: '.', k: solid}
	}
	for k := -2; k <= 2; k++ {
		az := v.center + float64(k)*45
		_, col := v.pos(0, az)
		g.put(row, col, compass8[int(norm360(az))/45])
	}
}
