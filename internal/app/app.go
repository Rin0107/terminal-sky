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
)

// Version は goreleaser が埋める。
var Version = "dev"

// Run は引数を受け取って結果を out に書く。
func Run(args []string, out io.Writer) error {
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
	_, err := fmt.Fprintln(out, "sky tonight を打つと、選んだ都市から今夜見える星空 (星・星座線・惑星・月) と見どころが端末に出て、画面と実際の空を見比べて星を探せる")
	return err
}
