// 守りの検査の共通部分 (依存なし)。このディレクトリは守りの部分。
import { readdirSync, readFileSync, statSync, existsSync } from "node:fs";
import { join } from "node:path";

export const wk = JSON.parse(readFileSync("wk.json", "utf8"));

export function walk(dir, exts) {
  if (!existsSync(dir)) return [];
  const out = [];
  for (const name of readdirSync(dir)) {
    if (name === "node_modules" || name.startsWith(".")) continue;
    const p = join(dir, name);
    if (statSync(p).isDirectory()) out.push(...walk(p, exts));
    else if (exts.some((e) => p.endsWith(e))) out.push(p);
  }
  return out;
}

export function fail(title, problems) {
  if (problems.length === 0) {
    console.log(`ok: ${title}`);
    return;
  }
  console.error(`NG: ${title}`);
  for (const p of problems) console.error(`  - ${p}`);
  process.exitCode = 1;
}
