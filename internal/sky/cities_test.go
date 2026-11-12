package sky

import (
	"math"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

// 内蔵の都市: 日本は都道府県庁の所在地 47 と、海外の主な都市。名前は打ちやすい英小文字 (と -)、重ならない。
func TestCityTable(t *testing.T) {
	keyRe := regexp.MustCompile(`^[a-z]+(-[a-z]+)*$`)
	keys, labels := map[string]bool{}, map[string]bool{}
	japan, abroad := 0, 0
	for i, c := range Cities {
		if !keyRe.MatchString(c.Key) || keys[c.Key] || labels[c.Label] || c.Label == "" {
			t.Fatalf("%d: %+v", i, c)
		}
		keys[c.Key], labels[c.Label] = true, true
		if !cityPlaced(c) {
			t.Fatalf("%s: 位置か時刻の帯 %+v", c.Key, c)
		}
		if c.Abroad {
			abroad++
		} else {
			japan++
		}
	}
	if japan != 47 || abroad < 1 || Cities[0].Key != "tokyo" {
		t.Fatalf("日本 %d 海外 %d 先頭 %s", japan, abroad, Cities[0].Key)
	}
}

// cityPlaced は都市の緯度・経度が地球の上にあり、日本の都市なら日本の中 (北緯 24〜46・東経 122〜146) で JST か。
func cityPlaced(c City) bool {
	onEarth := -90 <= c.LatDeg && c.LatDeg <= 90 && -180 <= c.LonDeg && c.LonDeg <= 180
	inJapan := 24 <= c.LatDeg && c.LatDeg <= 46 && 122 <= c.LonDeg && c.LonDeg <= 146 && c.Zone == jst
	return onEarth && (c.Abroad || inJapan)
}

// どの都市の時刻の帯も引ける (time/tzdata を埋め込んでいるので、端末の設定に頼らない)。夏時間も入る。
func TestCityZones(t *testing.T) {
	for _, c := range Cities {
		if c.Zone == nil {
			t.Fatalf("%s: 時刻の帯がない", c.Key)
		}
	}
	london, _ := FindCity("london")
	if _, off := time.Date(2026, 7, 1, 12, 0, 0, 0, london.Zone).Zone(); off != 3600 {
		t.Fatalf("ロンドンの夏時間 %d", off)
	}
	// 表の書き誤り (ない時刻の帯) は黙って nil にせず、止める
	defer func() {
		if r := recover(); r == nil || !strings.Contains(r.(string), "Nowhere/Atlantis") {
			t.Fatalf("ない時刻の帯で止まらない: %v", r)
		}
	}()
	zone("Nowhere/Atlantis")
	t.Fatal("ない時刻の帯を引けた")
}

// 外から来る --city の文字列: どんな文字列でも止まらず、候補は 3 つまでで、どれも一覧にある名前。
// 一覧にある名前は FindCity が見つけ、見つけた都市の名前か表示の名前が、打った文字列 (前後の空白と大文字を除いて) と同じ。
func FuzzSuggestCities(f *testing.F) {
	for _, s := range []string{"tokyo", "sapporro", "naga", "", " SAPPORO ", "札幌", "new-york", "\xff\xfe", strings.Repeat("a", 300)} {
		f.Add(s)
	}
	keys := CityKeys()
	f.Fuzz(func(t *testing.T, s string) {
		got := SuggestCities(s)
		if len(got) > maxSuggest {
			t.Fatalf("%q: %d 個", s, len(got))
		}
		for _, k := range got {
			if !slices.Contains(keys, k) {
				t.Fatalf("%q: 一覧にない候補 %q", s, k)
			}
		}
		k := strings.ToLower(strings.TrimSpace(s))
		if c, ok := FindCity(s); ok && c.Key != k && c.Label != k {
			t.Fatalf("%q: %+v", s, c)
		}
		if d := editDistance(k, "tokyo"); d < 0 || d > max(len([]rune(k)), 5) || d != editDistance("tokyo", k) {
			t.Fatalf("%q: 編集距離 %d", s, d)
		}
	})
}

// AC-5-2: sky now --city sapporo と --city naha で、同じ日時の北極星 (ポラリス) の高度が 15 度以上違う。
// 北極星の高度はおよそ緯度なので (札幌 北緯 43.1・那覇 北緯 26.2)、1 年の 1 日おきの 21 時のどれでも。
func TestAC_5_2_PolarisAltitudeByCity(t *testing.T) {
	sapporo, ok1 := FindCity("sapporo")
	naha, ok2 := FindCity("naha")
	if !ok1 || !ok2 {
		t.Fatal("AC-5-2: 札幌か那覇がない")
	}
	polaris := star(t, "ポラリス")
	n := 0
	for d := 0; d < 365; d += 2 {
		at := time.Date(2026, 1, 1, 21, 0, 0, 0, jst).AddDate(0, 0, d)
		a1, _ := Horizontal(polaris, sapporo, at)
		a2, _ := Horizontal(polaris, naha, at)
		if a1-a2 < 15 {
			t.Fatalf("AC-5-2: %v 札幌 %.1f 那覇 %.1f", at, a1, a2)
		}
		// 高度は緯度から 1 度以内 (北極星は天の北極から 0.7 度ほど)
		if math.Abs(a1-sapporo.LatDeg) > 1.5 || math.Abs(a2-naha.LatDeg) > 1.5 {
			t.Fatalf("AC-5-2: %v 札幌 %.1f 那覇 %.1f", at, a1, a2)
		}
		n++
	}
	if n != 183 {
		t.Fatal(n)
	}
}

func TestEditDistance(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{{"", "", 0}, {"a", "", 1}, {"", "abc", 3}, {"kitten", "sitting", 3}, {"sapporro", "sapporo", 1}, {"tokyo", "kyoto", 4}, {"札幌", "札幌", 0}, {"札", "幌", 1}} {
		if got := editDistance(c.a, c.b); got != c.want {
			t.Fatalf("editDistance(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
	// 全件: 内蔵の都市の名前のどの組でも、対称で、0 は同じときだけ、長さの差以上・長い方以下
	keys := CityKeys()
	for _, a := range keys {
		for _, b := range keys {
			d := editDistance(a, b)
			lo := len(a) - len(b)
			if lo < 0 {
				lo = -lo
			}
			if d != editDistance(b, a) || (d == 0) != (a == b) || d < lo || d > max(len(a), len(b)) {
				t.Fatalf("%s %s: %d", a, b, d)
			}
		}
	}
}

// 近い名前の候補: 打ち間違い (編集距離 2 以下) か、3 文字以上の頭が合う名前を、近い順に 3 つまで。
func TestSuggestCities(t *testing.T) {
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"sapporro", []string{"sapporo"}},
		{"Sapporo ", []string{"sapporo"}},                // 一覧にある名前は FindCity が先に見つけるが、候補にも出る
		{"sap", []string{"sapporo", "saga"}},             // 頭が合うものが先、次に打ち間違い
		{"ky", []string{}},                               // 2 文字は頭では選ばない (kyoto は編集距離 3 で遠い)
		{"kyo", []string{"kyoto", "tokyo"}},              // 3 文字なら頭で選ぶ (tokyo は編集距離 2)
		{"naga", []string{"nagano", "nagasaki", "nara"}}, // 頭が合う 2 つ、次に編集距離 1 (nara・naha・saga) を一覧の順で、3 つで切る
		{"newyork", []string{"new-york"}},
		{"atlantis", []string{}},
		{"", []string{}},
	} {
		if got := SuggestCities(c.in); !slices.Equal(got, c.want) {
			t.Fatalf("SuggestCities(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	// 全件: どの都市の名前を 1 文字消しても、候補は 3 つまでで、元の名前がそこに入る
	n := 0
	for _, k := range CityKeys() {
		for i := range k {
			typo := k[:i] + k[i+1:]
			got := SuggestCities(typo)
			if len(got) > maxSuggest || !slices.Contains(got, k) {
				t.Fatalf("%q (%s): %q", typo, k, got)
			}
			n++
		}
	}
	// 候補が多いとき (naga は近いものが 5 つ) は 3 つで切る (上の表)
	if n == 0 {
		t.Fatal(n)
	}
}
