// 受け入れ条件 (AC) ごとにテストがあるかの検査。
// 対象: 閉じた wk:task の issue と、この PR が閉じる issue。テストのどこかに "AC-<issue>-<n>" があること。
// CI でだけ動く (gh と GITHUB_TOKEN を使う)。手元では飛ばす。
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { fail, walk } from "./lib.mjs";

if (!process.env.GITHUB_ACTIONS) {
  console.log("skip: ac-coverage は CI でだけ動く");
  process.exit(0);
}
const repo = process.env.GITHUB_REPOSITORY;
const gh = (...a) => execFileSync("gh", a, { encoding: "utf8" });
const issues = JSON.parse(
  gh(
    "issue",
    "list",
    "--repo",
    repo,
    "--state",
    "closed",
    "--label",
    "wk:task",
    "--limit",
    "200",
    "--json",
    "number,body",
  ),
);

const ev = process.env.GITHUB_EVENT_PATH
  ? JSON.parse(readFileSync(process.env.GITHUB_EVENT_PATH, "utf8"))
  : {};
const prBody = ev.pull_request?.body ?? "";
for (const m of prBody.matchAll(/(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\s+#(\d+)/gi)) {
  const n = Number(m[1]);
  if (!issues.some((i) => i.number === n))
    issues.push(JSON.parse(gh("issue", "view", String(n), "--repo", repo, "--json", "number,body")));
}

const tests = walk(".", ["_test.go"])
  .map((f) => readFileSync(f, "utf8"))
  .join("\n");
const problems = [];
for (const is of issues) {
  for (const m of (is.body ?? "").matchAll(/^- \[[ x]\] AC-(\d+):/gm)) {
    const id = `AC-${is.number}-${m[1]}`;
    if (!tests.includes(id)) problems.push(`#${is.number} の ${id} のテストがない`);
  }
}
fail("受け入れ条件のテスト (ac-coverage)", problems);
