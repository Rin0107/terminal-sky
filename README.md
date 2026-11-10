# terminal-sky

今夜の星空をターミナルに描く Go の CLI。星座線・惑星・月齢・今夜の見どころ

## これは何
sky tonight を打つと、選んだ都市から今夜見える星空 (星・星座線・惑星・月) と見どころが端末に出て、画面と実際の空を見比べて星を探せる

## インストール
[Releases]((まだ公開していません)) から、自分の OS のファイルを取ってくる。または:

```bash
go install github.com/Rin0107/terminal-sky/cmd/app@latest
```

## 使い方
<!-- 30 秒でわかる使い方 -->

## データについて
このツールは重要な情報を持たず、外に何も送りません。詳しくは [DATA.md](DATA.md)。

## 開発
```bash
make verify   # gofmt・vet・lint・テスト・守りの検査・govulncheck・ビルド
```

---
[wakuwaku factory](https://github.com/Rin0107/ideas) が自動で作りました (アイディア: `00003-terminal-planetarium`)。
