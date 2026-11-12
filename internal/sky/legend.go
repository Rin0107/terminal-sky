package sky

import (
	"fmt"
	"strings"
)

// NothingUp は、地平線より上に月も惑星もないときの凡例 (幅 20 で、いちばん狭い図にも入る)。
const NothingUp = "月と惑星は地平線の下"

// MoonLegend は月の凡例 (記号・名前・月齢)。例: "D 月 (月齢 7.4)"。
func MoonLegend(m MoonSeen) string {
	return fmt.Sprintf("%c 月 (月齢 %.1f)", MoonSymbol(m.Age), m.Age)
}

// moonLegendShort は狭い端末での月の凡例。例: "D 月齢 7.4"。
func moonLegendShort(m MoonSeen) string {
	return fmt.Sprintf("%c 月齢 %.1f", MoonSymbol(m.Age), m.Age)
}

// Legend は図の下に出す 1 行の凡例 (幅 width 以下)。地平線より上の月 (あれば先に) と惑星を並べる。
// 入らなければ 3 段で縮める:
//  1. 記号と名前: "D 月 (月齢 7.4)  木 木星  土 土星"
//  2. 名前だけ (惑星の記号は名前の 1 文字目): "D 月齢 7.4 木星 土星"
//  3. 入る分だけ並べ、残りを数で示す: "D 月齢 7.4 木星 ほか 2"
//
// 星座線の文字 (- | / \) は使わない (図の線と見分けるため)。
func Legend(moon MoonSeen, planets []PlanetSeen, width int) string {
	var full, short []string
	if moon.Up() {
		full, short = append(full, MoonLegend(moon)), append(short, moonLegendShort(moon))
	}
	for _, p := range planets {
		full = append(full, fmt.Sprintf("%c %s", p.Planet.Symbol, p.Planet.Name))
		short = append(short, p.Planet.Name)
	}
	if len(full) == 0 {
		return NothingUp
	}
	if s := strings.Join(full, "  "); Width(s) <= width {
		return s
	}
	return fitWithRest(short, width)
}

// fitWithRest は items を前から、幅 width に入るだけ並べる。入らない分は "ほか <数>" で示す。
// 全部を落とした "ほか <数>" は幅 20 の図にも入る。
func fitWithRest(items []string, width int) string {
	if s := strings.Join(items, " "); Width(s) <= width {
		return s
	}
	for k := len(items) - 1; k > 0; k-- {
		if s := fmt.Sprintf("%s ほか %d", strings.Join(items[:k], " "), len(items)-k); Width(s) <= width {
			return s
		}
	}
	return fmt.Sprintf("ほか %d", len(items))
}
