# spec — 仕様

コードより先に、ここで「何が正しいか」を決める。

| ファイル | 道具 | 何を固定するか |
|---|---|---|
| `sky.smt2` | Z3 | 描画の算術の性質。性質の否定を置き、すべての (check-sat) が `unsat` |

| 性質 | 実装を確かめるテスト (`internal/sky/spec_test.go`) |
|---|---|
| 1. 地平線の楕円が、どの大きさの格子にも収まる | `TestSpec_Layout` (幅 20〜400 × 高さ 7〜199 の全件) |
| 2. 明るい星ほど目立つ文字 (● > * > +) | `TestSpec_SymbolMonotonic` (等級 -2〜3 を 0.01 刻みで、a ≤ b の全部のペア) |
| 3. 地平線より上の星は、楕円の中に置かれる | `TestSpec_ProjectInside` (高度・方位の格子 × 大きさ) |
| (描くのは地平線より上の星だけ) | `TestSpec_VisibleAboveHorizon` (目録 × 1 年の時刻) |

決まり:
- 振る舞いを変える PR は、先に spec を変える。spec と実装と TestSpec_ を同じ PR で
- 手元で確かめる: `z3 spec/sky.smt2` (全部 `unsat`)
