// Go の CLI の守りの検査 (依存なし)。このディレクトリは守りの部分。
// - go.mod に禁止モジュール (解析・課金・LLM・BaaS) がない、直接の依存が多すぎない
// - コードに許可していない外部の宛先がない (送信先・トラッカー)
// - 個人と結びつく情報の欄 (構造体のフィールド・JSON のキー) がない
// - シェルや curl を呼ばない
import { readFileSync } from "node:fs";
import { fail, walk, wk } from "./lib.mjs";

const problems = [];
const gomod = readFileSync("go.mod", "utf8");
for (const m of wk.deny_modules) if (gomod.includes(m)) problems.push(`go.mod: ${m} は使えない`);
const direct = gomod.split("\n").filter((l) => /^\s+[\w./-]+ v\S+\s*$/.test(l)).length;
if (direct > 8) problems.push(`直接の依存が ${direct} 個 (8 個まで)`);

const allow = new Set(["pkg.go.dev", "github.com", ...wk.egress_allow]);
const field = new RegExp(`(^|_)(${wk.forbidden_fields.join("|")})(_|$)`, "i");
for (const f of walk(".", [".go"])) {
  if (f.endsWith("_test.go")) continue;
  const src = readFileSync(f, "utf8");
  for (const m of src.matchAll(/https?:\/\/([a-z0-9.-]+\.[a-z]{2,})/gi)) {
    const host = m[1].toLowerCase();
    if (!allow.has(host)) problems.push(`${f}: 外部の宛先 ${host} (wk.json の egress_allow はユーザーだけが変える)`);
  }
  for (const m of src.matchAll(/^\s+(\w+)\s+[\w.*[\]]+\s+`json:"(\w+)/gm)) {
    const snake = m[1].replace(/([A-Z]+)([A-Z][a-z])/g, "$1_$2").replace(/([a-z0-9])([A-Z])/g, "$1_$2");
    if (field.test(snake) || field.test(m[2])) problems.push(`${f}: 欄 ${m[1]} は持ってはいけない情報`);
  }
  if (/"os\/exec"/.test(src) && /"(curl|wget|sh|bash)"/.test(src)) problems.push(`${f}: シェルや curl を呼ばない`);
}
fail("Go の守り (go-guard)", [...new Set(problems)]);
