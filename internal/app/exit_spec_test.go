package app

import (
	"errors"
	"fmt"
	"testing"
)

// TestSpec_ExitCode: 13. 成功は 0、引数の誤りは 2、それ以外の失敗は 1。包んでも (fmt.Errorf の %w) 変わらない。
func TestSpec_ExitCode(t *testing.T) {
	base := errors.New("x")
	cases := []struct {
		err  error
		want int
	}{
		{nil, ExitOK},
		{base, ExitFail},
		{fmt.Errorf("包む: %w", base), ExitFail},
		{usage(base), ExitUsage},
		{fmt.Errorf("包む: %w", usage(base)), ExitUsage},
		{usage(usage(base)), ExitUsage},
	}
	for i, c := range cases {
		if got := ExitCode(c.err); got != c.want {
			t.Fatalf("%d: ExitCode(%v) = %d, want %d", i, c.err, got, c.want)
		}
	}
	if ExitOK == ExitFail || ExitFail == ExitUsage || ExitOK == ExitUsage {
		t.Fatal("終了コードが重なる")
	}
}
