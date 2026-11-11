// Package app は terminal-sky の本体。main から切り離してテストできるようにする。
//
// 守ること: 重要な情報を持たない・送らない (DATA.md)。
// 保存するなら os.UserConfigDir() の下に、このツールの設定だけを、権限 0600 で。
package app

import (
	"errors"
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
		return errors.New("知らない引数: " + fs.Arg(0))
	}
	_, err := fmt.Fprintln(out, "sky tonight を打つと、選んだ都市から今夜見える星空 (星・星座線・惑星・月) と見どころが端末に出て、画面と実際の空を見比べて星を探せる\n\n使い方: sky now [--city tokyo] [--at 2026-01-15T21:00:00+09:00] [--width 80] [--height 24]")
	return err
}

// nowOptions は sky now の引数。
type nowOptions struct {
	city   sky.City
	at     time.Time
	width  int
	height int
}

func parseNow(args []string) (nowOptions, error) {
	fs := flag.NewFlagSet("now", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	city := fs.String("city", "tokyo", "観測地の都市")
	at := fs.String("at", "", "時刻 (RFC 3339)。省略すると今")
	width := fs.Int("width", 80, "描画の幅")
	height := fs.Int("height", 24, "描画の高さ")
	if err := fs.Parse(args); err != nil {
		return nowOptions{}, fmt.Errorf("引数: %w", err)
	}
	if fs.NArg() > 0 {
		return nowOptions{}, errors.New("知らない引数: " + fs.Arg(0))
	}
	c, ok := sky.FindCity(*city)
	if !ok {
		return nowOptions{}, fmt.Errorf("知らない都市: %q (選べるのは %s)", *city, strings.Join(sky.CityKeys(), ", "))
	}
	t := Now()
	if *at != "" {
		p, err := time.Parse(time.RFC3339, *at)
		if err != nil {
			return nowOptions{}, fmt.Errorf("--at は RFC 3339 で書く (例: 2026-01-15T21:00:00+09:00): %q", *at)
		}
		t = p
	}
	if y := t.UTC().Year(); y < MinYear || y > MaxYear {
		return nowOptions{}, fmt.Errorf("--at の年は %d〜%d (UT): %d", MinYear, MaxYear, y)
	}
	if *width < MinWidth || *width > MaxWidth {
		return nowOptions{}, fmt.Errorf("--width は %d〜%d: %d", MinWidth, MaxWidth, *width)
	}
	if *height < MinHeight || *height > MaxHeight {
		return nowOptions{}, fmt.Errorf("--height は %d〜%d: %d", MinHeight, MaxHeight, *height)
	}
	return nowOptions{city: c, at: t, width: *width, height: *height}, nil
}

// render は 1 行目に見出し、その下に空の図を描いた行を返す。
func render(o nowOptions) []string {
	seen := sky.Visible(sky.Catalog, o.city, o.at)
	head := fmt.Sprintf("%s %s  地平線より上の明るい星 %d 個", o.city.Label, o.at.In(o.city.Zone).Format("2006-01-02 15:04 MST"), len(seen))
	lines := []string{sky.Truncate(head, o.width)}
	return append(lines, sky.Chart(seen, o.width, o.height-1)...)
}

func runNow(args []string, out io.Writer) error {
	o, err := parseNow(args)
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, strings.Join(render(o), "\n")+"\n")
	return err
}
