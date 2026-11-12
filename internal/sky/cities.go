package sky

import (
	"sort"
	"strings"
	"time"
	_ "time/tzdata" // 海外の都市の時刻の帯 (夏時間を含む) を、端末の設定に頼らずに引く
)

// City は内蔵の観測地。利用者の現在地は受け取らず、ここから名前で選ぶだけ。
type City struct {
	Key    string         // --city で指定する名前
	Label  string         // 表示する名前
	LatDeg float64        // 緯度 [度] (北が正)
	LonDeg float64        // 経度 [度] (東が正)
	Zone   *time.Location // 表示に使う時刻の帯
	Abroad bool           // 海外の都市 (sky cities で日本の都市と分けて出す)
}

var jst = time.FixedZone("JST", 9*60*60)

// zone は時刻の帯を名前で引く (time/tzdata を埋め込んであるので、どの端末でも引ける)。
// 引けないのは表の書き誤りなので、黙って進まずに初期化のときに止める (TestCityZones が確かめる)。
func zone(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		panic("内蔵の都市の時刻の帯: " + err.Error())
	}
	return l
}

// jp は日本の都市 (時刻の帯は JST)。
func jp(key, label string, lat, lon float64) City {
	return City{Key: key, Label: label, LatDeg: lat, LonDeg: lon, Zone: jst}
}

// abroad は海外の都市。
func abroad(key, label string, lat, lon float64, tz string) City {
	return City{Key: key, Label: label, LatDeg: lat, LonDeg: lon, Zone: zone(tz), Abroad: true}
}

// Cities は内蔵の都市。先頭の東京が既定 (--city を省いたとき)。
// 日本は都道府県庁の所在地 (北から順。位置は都道府県庁のおよその緯度・経度。端末の 1 升は 2〜4 度なので 0.01 度で足りる)、
// 海外は時刻の帯と緯度がばらける主な都市を少し (南半球も入れて、北極星が見えない空も選べるように)。
var Cities = []City{
	jp("tokyo", "東京", 35.6895, 139.6917),
	jp("sapporo", "札幌", 43.0642, 141.3469),
	jp("aomori", "青森", 40.8244, 140.7400),
	jp("morioka", "盛岡", 39.7036, 141.1527),
	jp("sendai", "仙台", 38.2689, 140.8721),
	jp("akita", "秋田", 39.7186, 140.1024),
	jp("yamagata", "山形", 38.2404, 140.3633),
	jp("fukushima", "福島", 37.7503, 140.4676),
	jp("mito", "水戸", 36.3418, 140.4468),
	jp("utsunomiya", "宇都宮", 36.5658, 139.8836),
	jp("maebashi", "前橋", 36.3912, 139.0608),
	jp("saitama", "さいたま", 35.8570, 139.6489),
	jp("chiba", "千葉", 35.6051, 140.1233),
	jp("yokohama", "横浜", 35.4478, 139.6425),
	jp("niigata", "新潟", 37.9026, 139.0236),
	jp("toyama", "富山", 36.6953, 137.2113),
	jp("kanazawa", "金沢", 36.5947, 136.6256),
	jp("fukui", "福井", 36.0652, 136.2216),
	jp("kofu", "甲府", 35.6642, 138.5684),
	jp("nagano", "長野", 36.6513, 138.1810),
	jp("gifu", "岐阜", 35.3912, 136.7223),
	jp("shizuoka", "静岡", 34.9769, 138.3831),
	jp("nagoya", "名古屋", 35.1802, 136.9066),
	jp("tsu", "津", 34.7303, 136.5086),
	jp("otsu", "大津", 35.0045, 135.8686),
	jp("kyoto", "京都", 35.0214, 135.7556),
	jp("osaka", "大阪", 34.6863, 135.5200),
	jp("kobe", "神戸", 34.6913, 135.1830),
	jp("nara", "奈良", 34.6851, 135.8329),
	jp("wakayama", "和歌山", 34.2261, 135.1675),
	jp("tottori", "鳥取", 35.5036, 134.2383),
	jp("matsue", "松江", 35.4723, 133.0505),
	jp("okayama", "岡山", 34.6618, 133.9344),
	jp("hiroshima", "広島", 34.3966, 132.4596),
	jp("yamaguchi", "山口", 34.1859, 131.4714),
	jp("tokushima", "徳島", 34.0658, 134.5593),
	jp("takamatsu", "高松", 34.3401, 134.0434),
	jp("matsuyama", "松山", 33.8417, 132.7661),
	jp("kochi", "高知", 33.5597, 133.5311),
	jp("fukuoka", "福岡", 33.6064, 130.4181),
	jp("saga", "佐賀", 33.2494, 130.2988),
	jp("nagasaki", "長崎", 32.7448, 129.8737),
	jp("kumamoto", "熊本", 32.7898, 130.7417),
	jp("oita", "大分", 33.2382, 131.6126),
	jp("miyazaki", "宮崎", 31.9111, 131.4239),
	jp("kagoshima", "鹿児島", 31.5602, 130.5581),
	jp("naha", "那覇", 26.2124, 127.6809),
	abroad("london", "ロンドン", 51.5074, -0.1278, "Europe/London"),
	abroad("paris", "パリ", 48.8566, 2.3522, "Europe/Paris"),
	abroad("new-york", "ニューヨーク", 40.7128, -74.0060, "America/New_York"),
	abroad("honolulu", "ホノルル", 21.3069, -157.8583, "Pacific/Honolulu"),
	abroad("singapore", "シンガポール", 1.3521, 103.8198, "Asia/Singapore"),
	abroad("sydney", "シドニー", -33.8688, 151.2093, "Australia/Sydney"),
}

// FindCity は名前 (大文字小文字は区別しない) か、表示する名前 (「札幌」) で都市を探す。
func FindCity(key string) (City, bool) {
	k := strings.ToLower(strings.TrimSpace(key))
	for _, c := range Cities {
		if c.Key == k || c.Label == k {
			return c, true
		}
	}
	return City{}, false
}

// CityKeys は選べる都市の名前の一覧 (Cities の順)。
func CityKeys() []string {
	keys := make([]string, 0, len(Cities))
	for _, c := range Cities {
		keys = append(keys, c.Key)
	}
	return keys
}

// maxSuggest は誤った --city に出す候補の数の上限。
const maxSuggest = 3

// SuggestCities は一覧にない名前 key に近い都市の名前を、近い順 (同じなら一覧の順) に maxSuggest 個まで返す。
// 近いとは、3 文字以上打った頭が合う (「sap」→ sapporo。いちばん近いとみなす) か、打ち間違い (編集距離が 2 以下) のこと。
// 近いものがなければ空。
func SuggestCities(key string) []string {
	k := strings.ToLower(strings.TrimSpace(key))
	type cand struct {
		key  string
		dist int
	}
	var out []cand
	for _, c := range Cities {
		if len(k) >= 3 && strings.HasPrefix(c.Key, k) {
			out = append(out, cand{c.Key, 0})
		} else if d := editDistance(k, c.Key); d <= 2 {
			out = append(out, cand{c.Key, d})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].dist < out[j].dist })
	keys := []string{}
	for i := 0; i < len(out) && i < maxSuggest; i++ {
		keys = append(keys, out[i].key)
	}
	return keys
}

// editDistance は a と b の編集距離 (1 文字の足す・消す・置き換えるの回数の最小。文字は rune で数える)。
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}
