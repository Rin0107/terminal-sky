// Package app は terminal-sky の本体。main から切り離してテストできるようにする。
//
// 守ること: 重要な情報を持たない・送らない (DATA.md)。
// 保存するなら os.UserConfigDir() の下に、このツールの設定だけを、権限 0600 で。
package app

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Rin0107/terminal-sky/internal/sky"
)

// Version は goreleaser が埋める。
var Version = "dev"

// Now は今の時刻 (テストで差し替える)。
var Now = time.Now

// 描画の大きさの範囲。
const (
	MinWidth, MaxWidth   = 20, 400
	MinHeight, MaxHeight = 8, 200
	// 恒星時・歳差の式と、固有運動を入れない目録が十分に正しい年の範囲。
	MinYear, MaxYear = 1900, 2100
)

// Run は引数を受け取って結果を out に書く。
func Run(args []string, out io.Writer) error {
	if len(args) > 0 && args[0] == "now" {
		return runNow(args[1:], out)
	}
	if len(args) > 0 && args[0] == "tonight" {
		return runTonight(args[1:], out)
	}
	fs := flag.NewFlagSet("terminal-sky", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	showVersion := fs.Bool("version", false, "バージョンを表示する")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("引数: %w", err)
	}
	if *showVersion {
		_, err := fmt.Fprintln(out, Version)
		return err
	}
	if fs.NArg() > 0 {
		return unknownArg(fs.Arg(0))
	}
	_, err := fmt.Fprintln(out, "sky now を打つと、選んだ都市から今夜見える星空 (星・星座線・惑星・月) と見どころが端末に出て、画面と実際の空を見比べて星を探せる\n\n"+usageLine)
	return err
}

// usageLine は使い方の 1 行。余分な引数の誤りの文にも付ける (次に何を打てばよいか分かるように)。
const usageLine = "使い方: sky now [--city tokyo] [--at 2026-01-15T21:00:00+09:00] [--width 80] [--height 24] [--no-lines]" +
	" / sky tonight [--city tokyo] [--date 2026-01-15] [--width 80] [--height 24] [--no-lines]"

func unknownArg(a string) error {
	return fmt.Errorf("知らない引数: %s (%s)", a, usageLine)
}

// switchFlag は値を取らない旗 (--no-lines)。flag パッケージの英語の誤りの文
// (invalid boolean value ...) を出さないように、書き方の誤りは Set で覚えておき、Parse の後に決まりの形の文で返す。
type switchFlag struct {
	on  bool
	bad *string // 付けられた誤った値 (なければ nil)
}

func (s *switchFlag) IsBoolFlag() bool { return true }
func (s *switchFlag) String() string   { return "" }

// Set は値なし (flag が "true" を渡す) だけを受け付ける。
func (s *switchFlag) Set(v string) error {
	if v != "true" {
		s.bad = &v
		return nil
	}
	s.on = true
	return nil
}

// nowOptions は sky now の引数。
type nowOptions struct {
	city   sky.City
	at     time.Time
	width  int
	height int
	lines  bool // 星座線を描くか (--no-lines で消す)
}

// skyFlags は sky now と sky tonight に共通の引数。when は時刻の旗 (--at か --date) の文字列。
type skyFlags struct {
	fs                   *flag.FlagSet
	city, when           *string
	width, height        *int
	noLines              switchFlag
	whenName, whenSample string // 時刻の旗の名前と、書き方を誤ったときの文 (正しい書き方の例つき)
}

// newSkyFlags は共通の旗を登録する。when は時刻の旗の名前 (at・date)、sample は書き方を誤ったときの文。
func newSkyFlags(cmd, when, sample, usage string) *skyFlags {
	f := &skyFlags{fs: flag.NewFlagSet(cmd, flag.ContinueOnError), whenName: when, whenSample: sample}
	f.fs.SetOutput(io.Discard)
	f.city = f.fs.String("city", "tokyo", "観測地の都市")
	f.when = f.fs.String(when, "", usage)
	f.width = f.fs.Int("width", 80, "描画の幅")
	f.height = f.fs.Int("height", 24, "描画の高さ")
	f.fs.Var(&f.noLines, "no-lines", "星座線を描かない")
	return f
}

// parse は引数を読み、都市・大きさを確かめる (時刻は呼ぶ側で読む)。minHeight は --height の下限。
func (f *skyFlags) parse(args []string, minHeight int) (nowOptions, error) {
	err := f.fs.Parse(args)
	if f.noLines.bad != nil {
		return nowOptions{}, fmt.Errorf("--no-lines は値を付けずに書く (例: sky %s --no-lines): %q", f.fs.Name(), *f.noLines.bad)
	}
	if err != nil {
		return nowOptions{}, fmt.Errorf("引数: %w", err)
	}
	if f.fs.NArg() > 0 {
		return nowOptions{}, unknownArg(f.fs.Arg(0))
	}
	c, ok := sky.FindCity(*f.city)
	if !ok {
		return nowOptions{}, fmt.Errorf("知らない都市: %q (選べるのは %s)", *f.city, strings.Join(sky.CityKeys(), ", "))
	}
	if *f.width < MinWidth || *f.width > MaxWidth {
		return nowOptions{}, fmt.Errorf("--width は %d〜%d: %d", MinWidth, MaxWidth, *f.width)
	}
	if *f.height < minHeight || *f.height > MaxHeight {
		return nowOptions{}, fmt.Errorf("--height は %d〜%d: %d", minHeight, MaxHeight, *f.height)
	}
	return nowOptions{city: c, width: *f.width, height: *f.height, lines: !f.noLines.on}, nil
}

// whenAt は時刻の旗を layout で読む (読む時刻の帯は zone)。空なら今。
// 年は MinYear〜MaxYear で、zone の時刻の帯で数える (sky now は UT、sky tonight は都市の暦の日)。
func (f *skyFlags) whenAt(layout string, zone *time.Location) (time.Time, error) {
	t := Now()
	if *f.when != "" {
		p, err := time.ParseInLocation(layout, *f.when, zone)
		if err != nil {
			return time.Time{}, fmt.Errorf("%s: %q", f.whenSample, *f.when)
		}
		t = p
	}
	if y := t.In(zone).Year(); y < MinYear || y > MaxYear {
		return time.Time{}, fmt.Errorf("--%s の年は %d〜%d (%s): %d", f.whenName, MinYear, MaxYear, zoneName(zone), y)
	}
	return t, nil
}

// zoneName は年を数える時刻の帯の名前 (誤りの文に出す)。
func zoneName(zone *time.Location) string {
	if zone == time.UTC {
		return "UT"
	}
	return "都市の暦"
}

func parseNow(args []string) (nowOptions, error) {
	f := newSkyFlags("now", "at", "--at は RFC 3339 で書く (例: 2026-01-15T21:00:00+09:00)", "時刻 (RFC 3339)。省略すると今")
	o, err := f.parse(args, MinHeight)
	if err != nil {
		return nowOptions{}, err
	}
	if o.at, err = f.whenAt(time.RFC3339, time.UTC); err != nil {
		return nowOptions{}, err
	}
	return o, nil
}

// render は 1 行目に見出し、その下に空の図、月と惑星の凡例の 1 行、最後に notes の行を描いた行を返す
// (全部で o.height 行。図の高さは o.height - 2 - len(notes))。
func render(o nowOptions, head string, notes []string) []string {
	seen := sky.Visible(sky.Catalog, o.city, o.at)
	head = fmt.Sprintf("%s %s %s 地平線より上の明るい星 %d 個", o.city.Label, o.at.In(o.city.Zone).Format("2006-01-02 15:04 MST"), head, len(seen))
	lines := []string{sky.Truncate(head, o.width)}
	scene := sky.Scene{
		Stars:   seen,
		Figures: sky.Figures(sky.Constellations, sky.Stars(), o.city, o.at),
		NoLines: !o.lines,
		Moon:    sky.MoonAt(o.city, o.at),
		Planets: sky.VisiblePlanets(sky.Planets, o.city, o.at),
	}
	lines = append(lines, sky.Draw(scene, o.width, o.height-2-len(notes))...)
	lines = append(lines, sky.Legend(scene.Moon, scene.Planets, o.width))
	return append(lines, notes...) // notes は幅に収めてある (sky.Highlights)
}

func runNow(args []string, out io.Writer) error {
	o, err := parseNow(args)
	if err != nil {
		return err
	}
	return write(out, render(o, "", nil))
}

func write(out io.Writer, lines []string) error {
	_, err := io.WriteString(out, strings.Join(lines, "\n")+"\n")
	return err
}

// MinTonightHeight は sky tonight の --height の下限。見どころの 3 行を足しても、図の高さが 6 行 (sky now の下限と同じ) 残る。
const MinTonightHeight = MinHeight + 3

// tonightOptions は sky tonight の引数。sky now の引数の時刻の代わりに、暦の日を受け取る。
type tonightOptions struct {
	nowOptions
	day time.Time // 都市の時刻の帯での暦の日 (0 時)
}

func parseTonight(args []string) (tonightOptions, error) {
	f := newSkyFlags("tonight", "date", "--date は 年-月-日 で書く (例: 2026-01-15)", "暦の日 (年-月-日)。省略すると今日")
	o, err := f.parse(args, MinTonightHeight)
	if err != nil {
		return tonightOptions{}, err
	}
	t, err := f.whenAt(time.DateOnly, o.city.Zone)
	if err != nil {
		return tonightOptions{}, err
	}
	y, m, d := t.In(o.city.Zone).Date()
	return tonightOptions{nowOptions: o, day: time.Date(y, m, d, 0, 0, 0, 0, o.city.Zone)}, nil
}

// runTonight は日没の 1 時間後の空を描き、その下に今夜の見どころ (2〜3 行) を出す。全部で --height 行。
func runTonight(args []string, out io.Writer) error {
	o, err := parseTonight(args)
	if err != nil {
		return err
	}
	n, err := sky.Tonight(o.city, o.day)
	if err != nil {
		return fmt.Errorf("%w (別の --date か --city を選ぶ)", err) // 次にすること
	}
	// 日没と空を描く時刻は分に四捨五入する (暦の表と同じ。見比べたときに 1 分ずれないように)
	o.at = n.SkyAt.Round(time.Minute)
	head := fmt.Sprintf("(日没 %s の 1 時間後)", n.Sunset.Round(time.Minute).In(o.city.Zone).Format("15:04"))
	return write(out, render(o.nowOptions, head, sky.Highlights(n, o.width)))
}
