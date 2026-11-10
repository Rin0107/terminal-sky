// terminal-sky: 今夜の星空をターミナルに描く Go の CLI。星座線・惑星・月齢・今夜の見どころ
package main

import (
	"fmt"
	"os"

	"github.com/Rin0107/terminal-sky/internal/app"
)

func main() {
	if err := app.Run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "terminal-sky:", err)
		os.Exit(1)
	}
}
