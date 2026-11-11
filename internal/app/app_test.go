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
	c, _ := sky.FindCity("tokyo")
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
