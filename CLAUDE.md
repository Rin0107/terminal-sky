# terminal-sky — エージェントの決まり

wakuwaku factory が作る Go の CLI。工場の設計: `~/src/okuos/repos/ideas/docs/DESIGN.md`。

## 一番大事: 重要な情報を持たない
- アカウント・トークン・メールアドレス・名前・住所・IP・位置を**作らない・受け取らない・残さない・送らない**
- 保存するなら `os.UserConfigDir()` の下に、このツールの設定だけを権限 0600 で。保存する前に `DATA.md` を更新する
- 外部への通信は鍵の要らない公開 API だけ (wk.json の egress_allow はユーザーだけが変える)。解析・テレメトリは入れない
- シェルや curl を呼ばない。ファイルのパスは利用者の入力をそのまま使わず `filepath.Clean` して範囲を確かめる
- 秘密情報を repo に置かない

## 進め方
- 1 issue = 1 ブランチ = 1 PR (`Closes #N`)。main に直接 push しない
- issue の受け入れ条件ごとにテストを書く。テスト名かコメントに `AC-<issue>-<n>` を入れる (CI が検査)
- `make verify` が通ってから PR。`wk review` → `wk merge` (gh pr merge は使わない)
- 依存は標準ライブラリを優先。足すなら PR に理由を書く (直接の依存は 8 個まで)

## 変えてはいけないもの (guard が止める)
`.github/` `.claude/` `scripts/` `wk.json`
