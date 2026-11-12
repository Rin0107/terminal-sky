package app

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Rin0107/terminal-sky/internal/sky"
)

// AC-5-1: sky cities を実行すると、tokyo と sapporo と naha を含む一覧が出る。
// 1 行に 1 都市 (--city に書く名前・表示の名前・緯度) で、日本と海外に分ける。全部の都市が 1 度ずつ出る。
func TestAC_5_1_Cities(t *testing.T) {
	var b strings.Builder
	if err := Run([]string{"cities"}, &b); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	t.Log("\n" + out)
	lines, rows := cityRows(out)
	for _, k := range []string{"tokyo", "sapporo", "naha"} {
		if rows[k] == "" {
			t.Fatalf("AC-5-1: %s が一覧にない\n%s", k, out)
		}
	}
	if !strings.Contains(rows["sapporo"], "札幌") || !strings.Contains(rows["sapporo"], "北緯 43.1") || !strings.Contains(rows["sydney"], "南緯 33.9") {
		t.Fatalf("AC-5-1: 行の中身\n%s", out)
	}
	if len(rows) != len(sky.Cities) || len(lines) != len(sky.Cities)+3 || lines[1] != "日本" || lines[len(lines)-7] != "海外" {
		t.Fatalf("AC-5-1: %d 都市 / %d 行\n%s", len(rows), len(lines), out)
	}
	// 表示の名前の列がそろう (全角を 2 と数えて、緯度の列の始まりが同じ)
	cols := map[int]bool{}
	for _, l := range rows {
		cols[sky.Width(l[:strings.Index(l, "緯")])] = true
	}
	if len(cols) != 1 {
		t.Fatalf("AC-5-1: 列がそろわない %v\n%s", cols, out)
	}
}

// cityRows は sky cities の出力を行に分け、都市の行 (字下げ 2 つ・4 語) を --city の名前で引けるようにする。
func cityRows(out string) ([]string, map[string]string) {
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	rows := map[string]string{}
	for _, l := range lines {
		if f := strings.Fields(l); strings.HasPrefix(l, "  ") && len(f) == 4 {
			rows[f[0]] = l
		}
	}
	return lines, rows
}

// sky cities に余分な引数は引数の誤り (終了コード 2)。書けなければ 1。
func TestCitiesRejects(t *testing.T) {
	if err := Run([]string{"cities", "tokyo"}, &strings.Builder{}); ExitCode(err) != ExitUsage || !strings.Contains(err.Error(), "知らない引数: tokyo") {
		t.Fatal(err)
	}
	if err := Run([]string{"cities"}, failWriter{}); ExitCode(err) != ExitFail {
		t.Fatal(err)
	}
}

// polarisColumn は出力の中のポラリスの星の点 (名前の 2 つ左。名前は右に置く) の、端末での列。なければ -1。
func polarisColumn(out string) int {
	for _, l := range strings.Split(out, "\n") {
		if i := strings.Index(l, "+ ポラリス"); i >= 0 {
			return sky.Width(l[:i])
		}
	}
	return -1
}

// AC-5-3: --facing north を付けると、東京・2026-01-15T21:00+09:00 で北極星が出力の左右中央付近 (中央 3 分の 1) に描かれる。
// 既定の幅でも、狭い端末・広い端末でも。
func TestAC_5_3_FacingNorthPolaris(t *testing.T) {
	for _, w := range []int{80, 40, 120} {
		out := runNowOut(t, "--at", "2026-01-15T21:00:00+09:00", "--city", "tokyo", "--facing", "north", "--width", strconv.Itoa(w))
		if w == 80 {
			t.Log("\n" + out)
		}
		col := polarisColumn(out)
		if col < w/3 || col >= w-w/3 {
			t.Fatalf("AC-5-3: 幅 %d で北極星の列 %d (中央 3 分の 1 は %d〜%d)\n%s", w, col, w/3, w-w/3-1, out)
		}
		// 地平線の行は 左が西・真ん中が北・右が東
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		horizon := lines[len(lines)-2]
		if !strings.HasPrefix(horizon, "W") || !strings.HasSuffix(horizon, "E") || sky.Width(horizon[:strings.Index(horizon, "N.")]) != (w-1)/2 {
			t.Fatalf("AC-5-3: 地平線 %q", horizon)
		}
		// 見出しに向き。狭い端末では短く「北の空」にして、途中で切らない
		head := map[int]string{80: "東京 2026-01-15 21:00 JST 北を向いた空の明るい星 10 個", 40: "東京 2026-01-15 21:00 JST 北の空", 120: "東京 2026-01-15 21:00 JST 北を向いた空の明るい星 10 個"}[w]
		if lines[0] != head {
			t.Fatalf("AC-5-3: 見出し %q (want %q)", lines[0], head)
		}
	}
}

// --facing は 4 つの方角を受け付け (大文字小文字は区別しない)、sky tonight でも使える。それ以外は引数の誤り。
func TestFacingFlag(t *testing.T) {
	for _, c := range []struct {
		v    string
		want sky.Facing
	}{{"north", sky.North}, {"EAST", sky.East}, {"south", sky.South}, {"West", sky.West}} {
		o, err := parseNow([]string{"--facing", c.v})
		if err != nil || o.facing != c.want {
			t.Fatalf("%s: %+v %v", c.v, o, err)
		}
	}
	if o, err := parseNow(nil); err != nil || o.facing != sky.AllSky {
		t.Fatalf("既定は全天: %+v %v", o, err)
	}
	err := Run([]string{"now", "--facing", "up"}, &strings.Builder{})
	if err == nil || err.Error() != `--facing は north・east・south・west のどれか: "up"` || ExitCode(err) != ExitUsage {
		t.Fatal(err)
	}
}

// 南を向いた sky tonight: 見出しと見どころはそのまま、全部で --height 行。
func TestTonightFacing(t *testing.T) {
	out := runTonightOut(t, "--date", "2026-01-15", "--facing", "south")
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 24 || !strings.HasPrefix(lines[0], "東京 2026-01-15 17:51 JST (日没 16:51 の 1 時間後) 南を向いた空の明るい星 ") ||
		!strings.HasPrefix(lines[len(lines)-4], "E") || !strings.HasSuffix(lines[len(lines)-4], "W") || !strings.Contains(out, "一番明るい惑星は木星") {
		t.Fatalf("\n%s", out)
	}
	checkSize(t, out, 80, 24)
}

// 見出しの星の数は図に入る星だけ (背中側は数えない): 北と南の数を足すと、全天の数以上。
func TestFacingStarCount(t *testing.T) {
	count := func(args ...string) int {
		h := strings.SplitN(runNowOut(t, append([]string{"--at", "2026-01-15T21:00:00+09:00"}, args...)...), "\n", 2)[0]
		f := strings.Fields(h)
		n, err := strconv.Atoi(f[len(f)-2])
		if err != nil {
			t.Fatal(h)
		}
		return n
	}
	all, north, south := count(), count("--facing", "north"), count("--facing", "south")
	if north >= all || south >= all || north+south < all {
		t.Fatalf("全天 %d 北 %d 南 %d", all, north, south)
	}
}

// 見出しの星の数は、幅に入らなければ短い言い方にし、それでも入らなければ数を省く (途中で切らない)。
func TestHeaderShortens(t *testing.T) {
	for w, want := range map[int]string{
		60: "東京 2026-01-15 21:00 JST 地平線より上の明るい星 25 個",
		40: "東京 2026-01-15 21:00 JST 星 25 個",
		30: "東京 2026-01-15 21:00 JST",
		20: "東京 2026-01-15 21:0", // 日時も入らない幅だけは、幅で切る
	} {
		out := runNowOut(t, "--at", "2026-01-15T21:00:00+09:00", "--width", strconv.Itoa(w))
		if got := strings.SplitN(out, "\n", 2)[0]; got != want {
			t.Fatalf("幅 %d: %q (want %q)", w, got, want)
		}
	}
}

// AC-5-4: 一覧にない --city を渡すと、終了コード 2 で終わり、エラーに候補の都市名が出る。
// 近い名前があればそれを、なければ一覧の出し方と例 (tokyo) を出す。sky tonight でも同じ。
func TestAC_5_4_UnknownCity(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"now", "--city", "sapporro"}, `知らない都市: "sapporro" (近いのは sapporo。一覧は sky cities)`},
		{[]string{"now", "--city", "naga"}, `知らない都市: "naga" (近いのは nagano, nagasaki, nara。一覧は sky cities)`},
		{[]string{"tonight", "--city", "kyouto"}, `知らない都市: "kyouto" (近いのは kyoto。一覧は sky cities)`},
		{[]string{"now", "--city", "atlantis"}, `知らない都市: "atlantis" (一覧は sky cities。例: --city tokyo)`},
		{[]string{"now", "--city", ""}, `知らない都市: "" (一覧は sky cities。例: --city tokyo)`},
	} {
		err := Run(c.args, &strings.Builder{})
		if err == nil || err.Error() != c.want || ExitCode(err) != 2 {
			t.Fatalf("AC-5-4: %v: %v (終了コード %d, want %q)", c.args, err, ExitCode(err), c.want)
		}
	}
	// 一覧にある名前は、打ち方が違っても (大文字・表示の名前) 通る
	for _, k := range []string{"SAPPORO", "那覇"} {
		if _, err := parseNow([]string{"--city", k}); err != nil {
			t.Fatalf("%s: %v", k, err)
		}
	}
}

// 終了コード: 引数の誤りは 2 (打ち直せば通る)、引数は正しいができなかったときは 1、できたら 0。
func TestExitCodes(t *testing.T) {
	if ExitCode(nil) != ExitOK || usage(nil) != nil {
		t.Fatal("nil")
	}
	usageArgs := [][]string{
		{"nope"}, {"-nope"}, {"now", "extra"}, {"now", "--width", "19"}, {"now", "--height", "x"}, {"now", "--no-lines=maybe"},
		{"now", "--at", "2026-01-15 21:00"}, {"now", "--at", "2101-01-01T00:00:00Z"}, {"tonight", "--date", "2026/01/15"},
		{"tonight", "--height", "10"}, {"now", "--facing", "up"}, {"now", "--city", "x"}, {"cities", "x"},
	}
	for _, a := range usageArgs {
		err := Run(a, &strings.Builder{})
		if ExitCode(err) != ExitUsage {
			t.Fatalf("%v: %v → %d", a, err, ExitCode(err))
		}
		// 文はそのまま (包んでも変わらない)
		var u usageError
		if !errors.As(err, &u) || u.Error() != errors.Unwrap(err).Error() {
			t.Fatalf("%v: %v", a, err)
		}
	}
	// できなかった: 書けない・今夜の空を決められない
	if ExitCode(Run([]string{"now"}, failWriter{})) != ExitFail || ExitCode(errors.New("x")) != ExitFail {
		t.Fatal("書けない")
	}
	old := sky.Cities
	sky.Cities = append([]sky.City{{Key: "north", Label: "北の端", LatDeg: 78, LonDeg: 15, Zone: old[0].Zone}}, old...)
	t.Cleanup(func() { sky.Cities = old })
	if err := Run([]string{"tonight", "--city", "north", "--date", "2026-06-21"}, &strings.Builder{}); ExitCode(err) != ExitFail {
		t.Fatal(err)
	}
}

// どの内蔵の都市でも、sky now・sky tonight が --height 行・幅の中で出る (海外の夏時間の日も含めて、1 年の 15 日おき)。
func TestAllCitiesRender(t *testing.T) {
	n := 0
	for _, c := range sky.Cities {
		for d := 0; d < 365; d += 15 {
			date := time.Date(2026, 1, 1+d, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)
			out := runTonightOut(t, "--city", c.Key, "--date", date)
			if l := strings.Split(strings.TrimSuffix(out, "\n"), "\n"); len(l) != 24 || !strings.HasPrefix(out, c.Label+" ") {
				t.Fatalf("%s %s: %d 行\n%s", c.Key, date, len(l), out)
			}
			checkSize(t, out, 80, 24)
			n++
		}
	}
	if n != len(sky.Cities)*25 {
		t.Fatal(n)
	}
}
