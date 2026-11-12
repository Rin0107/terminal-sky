package app

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Rin0107/terminal-sky/internal/sky"
)

func TestRunVersion(t *testing.T) {
	var b strings.Builder
	if err := Run([]string{"-version"}, &b); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(b.String()) != Version {
		t.Fatalf("got %q", b.String())
	}
}

func TestRunRejectsUnknownArgs(t *testing.T) {
	if err := Run([]string{"nope"}, &strings.Builder{}); err == nil {
		t.Fatal("知らない引数を通した")
	}
	if err := Run([]string{"-nope"}, &strings.Builder{}); err == nil {
		t.Fatal("知らない旗を通した")
	}
}

func TestRunUsage(t *testing.T) {
	var b strings.Builder
	if err := Run(nil, &b); err != nil || !strings.Contains(b.String(), "sky now") {
		t.Fatalf("%v %q", err, b.String())
	}
}

func runNowOut(t *testing.T, args ...string) string {
	t.Helper()
	var b strings.Builder
	if err := Run(append([]string{"now"}, args...), &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// AC-1-1: sky now --at 2026-01-15T21:00:00+09:00 --city tokyo で、ベテルギウスとリゲルが地平線より上の星として出る。
func TestAC_1_1_OrionAboveHorizon(t *testing.T) {
	out := runNowOut(t, "--at", "2026-01-15T21:00:00+09:00", "--city", "tokyo")
	t.Log("\n" + out)
	for _, name := range []string{"ベテルギウス", "リゲル"} {
		if !strings.Contains(out, name) {
			t.Fatalf("AC-1-1: %s が出力にない", name)
		}
	}
	if !strings.HasPrefix(out, "東京 2026-01-15 21:00 JST") {
		t.Fatalf("AC-1-1: 見出し %q", strings.SplitN(out, "\n", 2)[0])
	}
	at, _ := time.Parse(time.RFC3339, "2026-01-15T21:00:00+09:00")
	c := tokyo(t)
	found := 0
	for _, s := range sky.Visible(sky.Catalog, c, at) {
		if s.Star.Name == "ベテルギウス" || s.Star.Name == "リゲル" {
			found++
		}
	}
	if found != 2 {
		t.Fatalf("AC-1-1: 地平線より上の星に 2 つともない (%d)", found)
	}
}

// AC-2-1: 東京・2026-01-15T21:00+09:00 で sky now を実行すると、出力に「オリオン」の名前が含まれる。
// 既定の大きさ (80 × 24) でも、広い端末でも。
func TestAC_2_1_OrionName(t *testing.T) {
	for _, args := range [][]string{
		{"--at", "2026-01-15T21:00:00+09:00", "--city", "tokyo"},
		{"--at", "2026-01-15T21:00:00+09:00", "--width", "120", "--height", "40"},
		{"--at", "2026-01-15T21:00:00+09:00", "--no-lines"},
	} {
		out := runNowOut(t, args...)
		if !strings.Contains(out, "オリオン") {
			t.Fatalf("AC-2-1: %v で「オリオン」がない\n%s", args, out)
		}
	}
}

// chart は見出しを除いた図の部分 (見出しの日付には '-' がある)。
func chart(out string) string {
	return strings.SplitN(out, "\n", 2)[1]
}

// AC-2-3: --no-lines を付けると星座線の文字が出力に含まれない。付けなければ含まれる。
func TestAC_2_3_NoLines(t *testing.T) {
	const lineChars = `-|/\`
	for _, at := range []string{"2026-01-15T21:00:00+09:00", "2026-08-01T21:00:00+09:00", "2026-04-20T03:00:00+09:00"} {
		for _, wh := range [][2]string{{"80", "24"}, {"120", "40"}, {"20", "8"}} {
			base := []string{"--at", at, "--width", wh[0], "--height", wh[1]}
			off := runNowOut(t, append(base, "--no-lines")...)
			if strings.ContainsAny(chart(off), lineChars) {
				t.Fatalf("AC-2-3: %v --no-lines で線の文字がある\n%s", base, off)
			}
			on := runNowOut(t, base...)
			if wh[0] != "20" && !strings.ContainsAny(chart(on), lineChars) {
				t.Fatalf("AC-2-3: %v で線がない\n%s", base, on)
			}
			// 線を消しても、星と星座の名前は同じだけ出る (線は星と名前に場所を譲るので)
			if strings.Count(on, "座") != strings.Count(off, "座") {
				t.Fatalf("AC-2-3: %v 星座の名前の数が変わった\n%s\n%s", base, on, off)
			}
		}
	}
}

func checkSize(t *testing.T, out string, w, h int) {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) > h {
		t.Fatalf("%d 行 (> %d)", len(lines), h)
	}
	for _, l := range lines {
		if len([]rune(l)) > w || sky.Width(l) > w {
			t.Fatalf("幅 %d/%d (> %d): %q", len([]rune(l)), sky.Width(l), w, l)
		}
	}
}

// AC-1-3: --width 60 --height 20 で、20 行以下・どの行も 60 文字以下 (全角は 2 と数えても)。
func TestAC_1_3_WidthHeight(t *testing.T) {
	out := runNowOut(t, "--at", "2026-01-15T21:00:00+09:00", "--width", "60", "--height", "20")
	t.Log("\n" + out)
	checkSize(t, out, 60, 20)
	// 境界値: 範囲の端の大きさでも守る
	for _, wh := range [][2]int{{MinWidth, MinHeight}, {MaxWidth, MaxHeight}, {MinWidth, MaxHeight}, {MaxWidth, MinHeight}} {
		out := runNowOut(t, "--at", "2026-01-15T21:00:00+09:00", "--width", strconv.Itoa(wh[0]), "--height", strconv.Itoa(wh[1]))
		checkSize(t, out, wh[0], wh[1])
	}
}

func TestNowDefaults(t *testing.T) {
	fixed := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	old := Now
	Now = func() time.Time { return fixed }
	t.Cleanup(func() { Now = old })
	o, err := parseNow(nil)
	if err != nil {
		t.Fatal(err)
	}
	if o.city.Key != "tokyo" || !o.at.Equal(fixed) || o.width != 80 || o.height != 24 || !o.lines {
		t.Fatalf("%+v", o)
	}
	out := runNowOut(t)
	if !strings.HasPrefix(out, "東京 2026-01-15 21:00 JST") {
		t.Fatal(out)
	}
	checkSize(t, out, 80, 24)
}

func TestNowRejects(t *testing.T) {
	bad := [][]string{
		{"--city", "atlantis"},
		{"--at", "2026-01-15 21:00"},
		{"--width", "19"},
		{"--width", "401"},
		{"--height", "7"},
		{"--height", "201"},
		{"--width", "x"},
		{"--no-lines=maybe"},
		{"extra"},
		// 境界の 1 つ外の年 (UT で数える)
		{"--at", "1899-12-31T23:59:59Z"},
		{"--at", "2101-01-01T00:00:00Z"},
		{"--at", "2100-12-31T23:30:00-01:00"}, // UT では 2101 年
	}
	for _, a := range bad {
		if err := Run(append([]string{"now"}, a...), &strings.Builder{}); err == nil {
			t.Fatalf("%v を通した", a)
		}
	}
	err := Run([]string{"now", "--city", "atlantis"}, &strings.Builder{})
	if !strings.Contains(err.Error(), "tokyo") {
		t.Fatal(err)
	}
	// 境界値: 端はそのまま通る
	for _, a := range [][]string{{"--width", "20"}, {"--width", "400"}, {"--height", "8"}, {"--height", "200"},
		{"--at", "1900-01-01T00:00:00Z"}, {"--at", "2100-12-31T23:59:59Z"}, {"--at", "1900-01-01T09:00:00+09:00"}} { // 最後は UT で 1900-01-01 0 時ちょうど
		if _, err := parseNow(a); err != nil {
			t.Fatalf("%v: %v", a, err)
		}
	}
}

// --no-lines の書き方の誤りは docs/design-system.md の「--no-lines」の文とそのまま同じ。余分な引数には使い方の 1 行を付ける
// (何が起きたか + 次にすること。flag パッケージの英語の文は出さない)。
func TestFlagErrorMessages(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"now", "--no-lines=maybe"}, `--no-lines は値を付けずに書く (例: sky now --no-lines): "maybe"`},
		{[]string{"now", "--no-lines=false"}, `--no-lines は値を付けずに書く (例: sky now --no-lines): "false"`},
		{[]string{"now", "-no-lines="}, `--no-lines は値を付けずに書く (例: sky now --no-lines): ""`},
		// 例は打ったコマンドに合わせる
		{[]string{"tonight", "--no-lines=maybe"}, `--no-lines は値を付けずに書く (例: sky tonight --no-lines): "maybe"`},
		{[]string{"tonight", "extra"}, `知らない引数: extra (` + usageLine + `)`},
		// 誤った --no-lines の後に別の誤りがあっても、先の --no-lines の文を返す
		{[]string{"now", "--no-lines=maybe", "--width", "x"}, `--no-lines は値を付けずに書く (例: sky now --no-lines): "maybe"`},
		{[]string{"now", "extra"}, `知らない引数: extra (` + usageLine + `)`},
		{[]string{"now", "--no-lines", "extra"}, `知らない引数: extra (` + usageLine + `)`},
		{[]string{"extra"}, `知らない引数: extra (` + usageLine + `)`},
	}
	for _, c := range cases {
		err := Run(c.args, &strings.Builder{})
		if err == nil || err.Error() != c.want {
			t.Errorf("%v: %v (want %q)", c.args, err, c.want)
		}
	}
	// 値なしの --no-lines と、flag が値なしと同じに扱う --no-lines=true は通る
	for _, a := range [][]string{{"--no-lines"}, {"-no-lines"}, {"--no-lines=true"}} {
		if o, err := parseNow(a); err != nil || o.lines {
			t.Fatalf("%v: %+v %v", a, o, err)
		}
	}
	var b strings.Builder
	if err := Run(nil, &b); err != nil || !strings.HasPrefix(b.String(), "sky now を打つと") || !strings.Contains(b.String(), usageLine) {
		t.Fatalf("使い方の文はコマンド名 sky now で始める: %v %q", err, b.String())
	}
	if (&switchFlag{}).String() != "" {
		t.Fatal("switchFlag の既定の表示は空")
	}
}

// tokyo は内蔵の東京。見つからなければテストを止める。
func tokyo(t *testing.T) sky.City {
	t.Helper()
	c, ok := sky.FindCity("tokyo")
	if !ok {
		t.Fatal("tokyo がない")
	}
	return c
}

// legendLine は出力の最後の行 (月と惑星の凡例)。
func legendLine(out string) string {
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	return lines[len(lines)-1]
}

// checkPlanetLegend は、凡例に地平線より下の惑星がないことと、地平線より上の惑星が名前で出る
// (all が false なら、入らない分は「ほか <数>」で数えられる) ことを確かめ、地平線より上の惑星の数を返す。
func checkPlanetLegend(t *testing.T, legend string, c sky.City, at time.Time, all bool) int {
	t.Helper()
	up, missing := 0, 0
	for _, p := range sky.Planets {
		alt, _ := sky.HorizontalPlanet(p, c, at)
		in := strings.Contains(legend, p.Name)
		if alt <= 0 && in || alt > 0 && !in && all {
			t.Fatalf("AC-3-3: %v の %s (高度 %.1f) と凡例 %q", at, p.Name, alt, legend)
		}
		if alt > 0 {
			up++
		}
		if alt > 0 && !in {
			missing++
		}
	}
	if missing > 0 && !strings.Contains(legend, "ほか ") {
		t.Fatalf("AC-3-3: %v の凡例 %q で惑星が %d 個、黙って消えた", at, legend, missing)
	}
	return up
}

// AC-3-3: 地平線より上にある惑星は、出力の凡例に名前が出る (地平線より下の惑星は出ない)。
// 1 年の 21 時と 3 時を 4 日おきに、既定の大きさと狭い端末 (幅 40 は名前だけに縮めて全部入る) で。
// いちばん狭い幅 20 では入らない分を「ほか <数>」で数え、黙って消さない。
func TestAC_3_3_PlanetsInLegend(t *testing.T) {
	c := tokyo(t)
	up := 0
	for d := 0; d < 365*2; d += 4 {
		at := time.Date(2026, 1, 1, 21, 0, 0, 0, c.Zone).AddDate(0, 0, d/2).Add(time.Duration(d%2) * 6 * time.Hour)
		for _, wh := range [][2]string{{"80", "24"}, {"40", "8"}, {"20", "8"}} {
			out := runNowOut(t, "--at", at.Format(time.RFC3339), "--width", wh[0], "--height", wh[1])
			lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
			up += checkPlanetLegend(t, legendLine(out), c, at, wh[0] != "20")
			// 凡例は図の下の 1 行で、全部で --height 行
			if h, _ := strconv.Atoi(wh[1]); len(lines) != h {
				t.Fatalf("%v: %d 行", wh, len(lines))
			}
		}
	}
	if up == 0 {
		t.Fatal("AC-3-3: 惑星が 1 度も地平線より上にない")
	}
	// 2026-01-15 21:00 の東京では木星が東の空に高い
	if out := runNowOut(t, "--at", "2026-01-15T21:00:00+09:00"); !strings.Contains(legendLine(out), "木 木星") || !strings.ContainsRune(chart(out), '木') {
		t.Fatalf("AC-3-3: 木星がない\n%s", out)
	}
}

// AC-3-4: 月が地平線より上にあるとき、出力に月の記号と月齢が表示される。
// 地平線より下のときは、図にも凡例にも月を出さない。
func TestAC_3_4_MoonSymbolAndAge(t *testing.T) {
	c := tokyo(t)
	up, down := 0, 0
	for d := 0; d < 60; d++ {
		at := time.Date(2026, 1, 1, 21, 0, 0, 0, c.Zone).AddDate(0, 0, d)
		out := runNowOut(t, "--at", at.Format(time.RFC3339))
		m := sky.MoonAt(c, at)
		legend := legendLine(out)
		if !m.Up() {
			down++
			if strings.Contains(legend, "月齢") {
				t.Fatalf("%v: 地平線の下の月が凡例にある %q", at, legend)
			}
			continue
		}
		up++
		sym := sky.MoonSymbol(m.Age)
		age := strconv.FormatFloat(m.Age, 'f', 1, 64)
		if !strings.Contains(legend, string(sym)+" 月 (月齢 "+age+")") {
			t.Fatalf("AC-3-4: %v 凡例 %q (月齢 %s)", at, legend, age)
		}
		graph := strings.Join(strings.Split(chart(out), "\n")[:22], "\n")
		if !strings.ContainsRune(graph, sym) {
			t.Fatalf("AC-3-4: %v 図に月の記号 %c がない\n%s", at, sym, out)
		}
	}
	if up == 0 || down == 0 {
		t.Fatal(up, down)
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("書けない") }

func TestWriteErrors(t *testing.T) {
	for _, a := range [][]string{nil, {"-version"}, {"now", "--at", "2026-01-15T21:00:00+09:00"}} {
		if err := Run(a, failWriter{}); err == nil {
			t.Fatalf("%v: 書けないのに成功した", a)
		}
	}
}

// AC-1-5: DATA.md に「保存するものはない・通信しない・現在地は受け取らない」と書かれている。
func TestAC_1_5_DataMd(t *testing.T) {
	b, err := os.ReadFile("../../DATA.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"保存するものはない", "通信しない", "現在地は受け取らない"} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("AC-1-5: DATA.md に %q がない", want)
		}
	}
}

func runTonightOut(t *testing.T, args ...string) string {
	t.Helper()
	var b strings.Builder
	if err := Run(append([]string{"tonight"}, args...), &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// highlightLines は sky tonight の出力の見どころの行 (凡例の行より後)。
func highlightLines(t *testing.T, out string) []string {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "一番明るい惑星は") || strings.HasPrefix(l, "今夜は") {
			return lines[i:]
		}
	}
	t.Fatalf("見どころがない\n%s", out)
	return nil
}

// AC-4-1: sky tonight --date 2026-01-15 --city tokyo を実行すると、星空の絵と 1〜3 行の見どころが出力される。
func TestAC_4_1_Tonight(t *testing.T) {
	out := runTonightOut(t, "--date", "2026-01-15", "--city", "tokyo")
	t.Log("\n" + out)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 24 {
		t.Fatalf("AC-4-1: %d 行 (--height 24)", len(lines))
	}
	// 星空の絵: 日没 (16:51、国立天文台の暦と同じ分) の 1 時間後の空。東の空に木星、オリオン座が昇ってくる
	if !strings.HasPrefix(out, "東京 2026-01-15 17:51 JST (日没 16:51 の 1 時間後)") || !strings.ContainsRune(chart(out), '木') ||
		!strings.Contains(out, "オリオン") {
		t.Fatalf("AC-4-1: 星空の絵がない\n%s", out)
	}
	hl := highlightLines(t, out)
	if len(hl) < 1 || len(hl) > 3 || !strings.Contains(hl[1], "月齢") {
		t.Fatalf("AC-4-1: 見どころ %q", hl)
	}
	checkSize(t, out, 80, 24)
	// 境界値: いちばん小さい高さ・幅でも、見どころまで --height 行に入る
	for _, wh := range [][2]int{{MinWidth, MinTonightHeight}, {MaxWidth, MaxHeight}, {40, MinTonightHeight}} {
		out := runTonightOut(t, "--date", "2026-09-08", "--width", strconv.Itoa(wh[0]), "--height", strconv.Itoa(wh[1]))
		if n := len(strings.Split(strings.TrimSuffix(out, "\n"), "\n")); n != wh[1] {
			t.Fatalf("AC-4-1: %v で %d 行", wh, n)
		}
		checkSize(t, out, wh[0], wh[1])
	}
}

// AC-4-2: 見どころの行に、方角が 16 方位の日本語 (例: 南東) で書かれている。1 年の 7 日おきの夜で。
func TestAC_4_2_CompassInHighlights(t *testing.T) {
	points := []string{"北北東", "東北東", "東南東", "南南東", "南南西", "西南西", "西北西", "北北西", "北東", "南東", "南西", "北西", "北", "東", "南", "西"}
	c := tokyo(t)
	for d := 0; d < 365; d += 7 {
		day := time.Date(2026, 1, 1, 0, 0, 0, 0, c.Zone).AddDate(0, 0, d).Format(time.DateOnly)
		for _, l := range highlightLines(t, runTonightOut(t, "--date", day, "--width", "120")) {
			if !strings.Contains(l, "の空") {
				continue
			}
			ok := false
			for _, p := range points {
				ok = ok || strings.Contains(l, p+"の空")
			}
			if !ok {
				t.Fatalf("AC-4-2: %s の %q に 16 方位がない", day, l)
			}
		}
	}
	if hl := highlightLines(t, runTonightOut(t, "--date", "2026-01-15")); !strings.Contains(hl[0], "南の空") {
		t.Fatalf("AC-4-2: %q", hl[0])
	}
}

func TestTonightDefaultsAndRejects(t *testing.T) {
	fixed := time.Date(2026, 1, 15, 3, 0, 0, 0, time.UTC) // 東京では 2026-01-15 12:00
	old := Now
	Now = func() time.Time { return fixed }
	t.Cleanup(func() { Now = old })
	if out := runTonightOut(t); !strings.HasPrefix(out, "東京 2026-01-15 17:51 JST") {
		t.Fatal(out)
	}
	bad := [][]string{
		{"--date", "2026/01/15"},
		{"--date", "2026-01-15T21:00:00+09:00"},
		{"--height", strconv.Itoa(MinTonightHeight - 1)},
		{"--city", "atlantis"},
		{"--at", "2026-01-15T21:00:00+09:00"}, // 時刻の旗は sky now だけ
		// 境界の 1 つ外の年 (都市の暦の日で数える)
		{"--date", "1899-12-31"},
		{"--date", "2101-01-01"},
		{"extra"},
	}
	for _, a := range bad {
		if err := Run(append([]string{"tonight"}, a...), &strings.Builder{}); err == nil {
			t.Fatalf("%v を通した", a)
		}
	}
	err := Run([]string{"tonight", "--date", "2026/01/15"}, &strings.Builder{})
	if err == nil || err.Error() != `--date は 年-月-日 で書く (例: 2026-01-15): "2026/01/15"` {
		t.Fatal(err)
	}
	err = Run([]string{"tonight", "--date", "2101-01-01"}, &strings.Builder{})
	if err == nil || err.Error() != "--date の年は 1900〜2100 (都市の暦): 2101" {
		t.Fatal(err)
	}
	for _, a := range [][]string{{"--date", "1900-01-01"}, {"--date", "2100-12-31"}, {"--height", strconv.Itoa(MinTonightHeight)}} {
		o, err := parseTonight(a)
		if err != nil {
			t.Fatalf("%v: %v", a, err)
		}
		if o.day.Hour() != 0 || o.day.Location() != o.city.Zone {
			t.Fatalf("%v: %v", a, o.day)
		}
	}
	if err := Run([]string{"tonight", "--date", "2026-01-15"}, failWriter{}); err == nil {
		t.Fatal("書けないのに成功した")
	}
}

// 内蔵の都市はどれも毎日日が沈むが、白夜の都市が足されても、止まらずに誤りの文を返す。
func TestTonightNoNight(t *testing.T) {
	old := sky.Cities
	sky.Cities = append([]sky.City{{Key: "north", Label: "北の端", LatDeg: 78, LonDeg: 15, Zone: time.UTC}}, old...)
	t.Cleanup(func() { sky.Cities = old })
	err := Run([]string{"tonight", "--city", "north", "--date", "2026-06-21"}, &strings.Builder{})
	// 何が起きたかと、次にすること (別の日か都市を選ぶ)
	if !errors.Is(err, sky.ErrNoNight) || !strings.Contains(err.Error(), "別の --date か --city を選ぶ") {
		t.Fatal(err)
	}
}

func FuzzRunTonight(f *testing.F) {
	f.Add("tokyo", "2026-01-15", 80, 24)
	f.Add("TOKYO", "1900-01-01", 20, 11)
	f.Add("", "", 400, 200)
	f.Fuzz(func(t *testing.T, city, date string, w, h int) {
		if w > MaxWidth || h > MaxHeight {
			return
		}
		args := []string{"tonight", "--city", city, "--width", strconv.Itoa(w), "--height", strconv.Itoa(h)}
		if date != "" {
			args = append(args, "--date", date)
		}
		var b strings.Builder
		if err := Run(args, &b); err != nil {
			return
		}
		checkSize(t, b.String(), w, h)
	})
}

func FuzzRunNow(f *testing.F) {
	f.Add("tokyo", "2026-01-15T21:00:00+09:00", 80, 24)
	f.Add("TOKYO", "1999-12-31T23:59:59Z", 20, 8)
	f.Add("", "", 400, 200)
	f.Fuzz(func(t *testing.T, city, at string, w, h int) {
		if w > MaxWidth || h > MaxHeight {
			return
		}
		args := []string{"now", "--city", city, "--width", strconv.Itoa(w), "--height", strconv.Itoa(h)}
		if at != "" {
			args = append(args, "--at", at)
		}
		var b strings.Builder
		if err := Run(args, &b); err != nil {
			return
		}
		checkSize(t, b.String(), w, h)
	})
}
