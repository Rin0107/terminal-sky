# spec — 仕様

コードより先に、ここで「何が正しいか」を決める。

| ファイル | 道具 | 何を固定するか |
|---|---|---|
| `sky.smt2` | Z3 | 描画の算術の性質。性質の否定を置き、すべての (check-sat) が `unsat` |

| 性質 | 実装を確かめるテスト (1〜6 は `internal/sky/spec_test.go`、7・8 と惑星は `internal/sky/planet_spec_test.go`、9・10 と今夜は `internal/sky/tonight_spec_test.go`) |
|---|---|
| 1. 地平線の楕円が、どの大きさの格子にも収まる | `TestSpec_Layout` (幅 20〜400 × 高さ 6〜199 の全件) |
| 2. 明るい星ほど目立つ文字 (● > * > +) | `TestSpec_SymbolMonotonic` (等級 -2〜3 を 0.01 刻みで、a ≤ b の全部のペア) |
| 3. 地平線より上の星は、楕円の中に置かれる | `TestSpec_ProjectInside` (高度・方位の格子 × 大きさ) |
| (描くのは地平線より上の星だけ) | `TestSpec_VisibleAboveHorizon` (目録 × 1 年の時刻) |
| 4. 両端とも地平線より下の星座線は描かない | `TestSpec_SegmentShown` (全部の星座線 × 1 年の時刻) |
| 5. 星座線の点は地平線の内側だけで、格子に収まる | `TestSpec_LineInsideHorizon` (格子の外まで広げた全部の升 × 大きさ) |
| 6. 星座線の文字は見た目の傾きと合う | `TestSpec_LineGlyph` (行・列のずれ -60〜60 の全部) |
| 7. 月齢は 0 以上・朔望月未満で、相は 0〜7 (新月 0・満月 4) | `TestSpec_MoonAge` (離角 0〜360 の 0.01 度刻みの全部) |
| 8. 月・惑星の記号は、投影した列か 1 つ左に必ず収まる | `TestSpec_BodyFits` (幅 20〜400 × 全部の列 × 幅 1・2) |
| (描く惑星は地平線より上だけで、凡例に全部出る) | `TestSpec_PlanetsAboveHorizon` (5 惑星 × 1 年の時刻) |
| (ケプラーの方程式の解は十分に正しい) | `TestSpec_Kepler` (離心率 0〜0.25 × 平均近点角の格子と PBT) |
| 9. 16 方位の名前は、方位にいちばん近い 16 方位 (ずれ 11.25 度まで) | `TestSpec_Compass16` (方位 -720〜720 の 0.01 度刻みの全部) |
| 10. 見どころは 1〜3 行 | `TestSpec_HighlightLines` (東京の 1 年の全部の夜) |
| (日没は日の出の前の日の入りで、太陽の高度がちょうど地平線の値) | `TestSpec_SunsetCrossing` (東京の 1 年の全部の日) |
| (接近と数えるのは、月と惑星が両方地平線より上で 5 度以内のときだけ) | `TestSpec_ApproachVisible` (1 年の夜の全部の時刻) |
| 11. 方角を向いた図に入るもの (高度 0 より上・向きから左右 90 度) は、地平線の行より上で左右の縁の中に置かれる | `TestSpec_FacingInside` (`internal/sky/facing_test.go`。4 つの向き × 高度 0.25 度・角 0.5 度刻み × 大きさ) |
| 12. 向きから見た角は -180〜180 未満で、差が 180 を超える 2 点の線は背中側を通る (描かない) | `TestSpec_RelAz` (`internal/sky/facing_test.go`。4 つの向き × 方位 -360〜720 の 0.25 度刻み) |
| 13. 終了コードは 成功 0・引数の誤り 2・それ以外の失敗 1 | `TestSpec_ExitCode` (`internal/app/exit_spec_test.go`) |

決まり:
- 振る舞いを変える PR は、先に spec を変える。spec と実装と TestSpec_ を同じ PR で
- 手元で確かめる: `z3 spec/sky.smt2` (全部 `unsat`)
