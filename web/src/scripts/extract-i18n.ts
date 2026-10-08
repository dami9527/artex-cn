/**
 * Script: extract-i18n.ts
 *
 * ARTEX —— UI 字符串抽取器。
 *
 * 用 TypeScript AST 解析 `src/` 下所有 `.ts`/`.tsx`，只挑出面向用户、
 * 硬编码在代码里的中文字符串。抽取对象有三类：
 *   1) 字符串字面量            例) title: "仪表盘"
 *   2) 模板字面量              例) `已删除 ${n} 个对话` → "已删除 {var0} 个对话"
 *   3) JSX 文本节点            例) <span>故障转移</span>
 * 注释(`//`, `/* *\/`)不是 AST 节点，因此天然被排除在外。写在语句前后注释里的
 * 中文不属于抽取对象。
 *
 * 产物(web/messages/)：
 *   - zh.json          按命名空间嵌套的 { 键: "中文原文" }
 *   - zh.sources.json  每个键的来源(文件:行号)·类型·占位符 — 供接线、翻译与漂移追踪使用
 *
 * 重新抽取是非破坏性的。只更新抽取产生的顶层命名空间(基于文件路径：
 * app·components·lib 等)，其余“人工整理命名空间”(手工维护的 nav·search·header 等)
 * 原样保留。规则：抽取命名空间只是清单，不手工修改；运行时文案只放在人工整理
 * 命名空间里。这样重新抽取才不会覆盖译文。
 *
 * 命名空间取自文件路径。例) src/app/(main)/system/llm/page.tsx
 *   → app.main.system.llm.page  (去掉路由分组括号、去掉 _文件夹的下划线)
 * 键是原文 sha1 的前 8 位。顺序或文件发生变化时，同一句话仍得到同一个键，
 * 重复执行的 diff 很小(便于对照上游更新)。
 *
 * 执行：npm run extract:i18n
 */

import * as ts from "typescript";

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";

const SRC_DIR = path.resolve(__dirname, "..");
const MESSAGES_DIR = path.resolve(__dirname, "../../messages");
const REPO_WEB_DIR = path.resolve(__dirname, "../..");

/** 是否包含至少一个汉字(Han)。判断是否为 UI 文案的唯一标准。 */
const HAN = /\p{Script=Han}/u;

/** 抽取时排除的目录(解析工具自身与构建产物)。 */
const SKIP_DIRS = new Set(["scripts", "node_modules", ".next"]);

type Kind = "string" | "template" | "jsx";

interface Message {
  ns: string;
  key: string;
  text: string;
  kind: Kind;
  placeholders: string[]; // 模板中 ${...} 的原始表达式(仅供参考)
  occurrences: string[]; // "相对路径:行号"
}

/** 以 src 为基准的相对路径 → 用点连接的命名空间。 */
function toNamespace(absFile: string): string {
  const rel = path.relative(SRC_DIR, absFile).replace(/\.[tj]sx?$/, "");
  return rel
    .split(path.sep)
    .map((seg) =>
      seg
        .replace(/^\((.*)\)$/, "$1") // (main) → main  (路由分组)
        .replace(/^_/, "") // _components → components
        .replace(/[^A-Za-z0-9\u4e00-\u9fff]+/g, "-")
        .replace(/^-+|-+$/g, ""),
    )
    .filter(Boolean)
    .join(".");
}

function hashKey(text: string): string {
  return crypto.createHash("sha1").update(text).digest("hex").slice(0, 8);
}

/** 递归遍历目录，收集 .ts/.tsx 文件。 */
function collectFiles(dir: string, out: string[] = []): string[] {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.isDirectory()) {
      if (SKIP_DIRS.has(entry.name)) continue;
      collectFiles(path.join(dir, entry.name), out);
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      out.push(path.join(dir, entry.name));
    }
  }
  return out;
}

/** 把模板字面量重组成「字面量片段 + {varN}」。返回 null = 字面量片段里没有汉字。 */
function reconstructTemplate(node: ts.TemplateExpression): { text: string; placeholders: string[] } | null {
  let text = node.head.text;
  const placeholders: string[] = [];
  node.templateSpans.forEach((span, i) => {
    placeholders.push(span.expression.getText());
    text += `{var${i}}${span.literal.text}`;
  });
  // 汉字若只出现在 ${} 内部、固定文案里没有，那些汉字会作为内部节点单独捕获。
  if (!HAN.test(node.head.text) && !node.templateSpans.some((s) => HAN.test(s.literal.text))) {
    return null;
  }
  return { text, placeholders };
}

/** JSX 文本归一化：去掉首尾空白 + 把内部连续空白/换行压成一个空格(与浏览器渲染规则一致)。 */
function normalizeJsxText(raw: string): string {
  return raw.replace(/\s+/g, " ").trim();
}

const messages = new Map<string, Message>(); // "ns\u0000key" → Message
const scannedFiles = collectFiles(SRC_DIR).sort();
const capturedLines = new Map<string, Set<number>>(); // 文件 → 命中抽取节点的行号集合
let fileWithHan = 0;

function record(ns: string, text: string, kind: Kind, placeholders: string[], relFile: string, line: number) {
  const key = hashKey(text);
  const id = `${ns}\u0000${key}`;
  let msg = messages.get(id);
  if (!msg) {
    msg = { ns, key, text, kind, placeholders, occurrences: [] };
    messages.set(id, msg);
  }
  const occ = `${relFile}:${line}`;
  if (!msg.occurrences.includes(occ)) msg.occurrences.push(occ);
}

for (const absFile of scannedFiles) {
  const source = fs.readFileSync(absFile, "utf8");
  if (!HAN.test(source)) continue;
  fileWithHan++;

  const relFile = path.relative(REPO_WEB_DIR, absFile);
  const ns = toNamespace(absFile);
  const scriptKind = absFile.endsWith(".tsx") ? ts.ScriptKind.TSX : ts.ScriptKind.TS;
  const sf = ts.createSourceFile(absFile, source, ts.ScriptTarget.Latest, true, scriptKind);
  const lineOf = (pos: number) => sf.getLineAndCharacterOfPosition(pos).line + 1;
  const markLine = (pos: number) => {
    const set = capturedLines.get(relFile) ?? new Set<number>();
    set.add(lineOf(pos));
    capturedLines.set(relFile, set);
  };

  const visit = (node: ts.Node): void => {
    if (ts.isStringLiteralLike(node) && !ts.isTemplateExpression(node.parent)) {
      // 字符串字面量 + 无替换的模板。(带替换的模板的 head/middle 不会走到这里)
      if (HAN.test(node.text)) {
        record(ns, node.text, "string", [], relFile, lineOf(node.getStart(sf)));
        markLine(node.getStart(sf));
      }
    } else if (ts.isTemplateExpression(node)) {
      const r = reconstructTemplate(node);
      if (r) {
        record(ns, r.text, "template", r.placeholders, relFile, lineOf(node.getStart(sf)));
        markLine(node.getStart(sf));
      }
    } else if (ts.isJsxText(node)) {
      const text = normalizeJsxText(node.text);
      if (text && HAN.test(text)) {
        record(ns, text, "jsx", [], relFile, lineOf(node.getStart(sf)));
        markLine(node.getStart(sf));
      }
    }
    ts.forEachChild(node, visit);
  };
  visit(sf);
}

// --- 组装嵌套 JSON ---------------------------------------------------------
type Tree = { [k: string]: Tree | string };

function setNested(root: Tree, nsPath: string, key: string, value: string) {
  const segs = nsPath.split(".");
  let node: Tree = root;
  for (const seg of segs) {
    if (typeof node[seg] !== "object") node[seg] = {};
    node = node[seg] as Tree;
  }
  node[key] = value;
}

/** 递归排序键，把重复执行的 diff 降到最小。 */
function sortTree(t: Tree): Tree {
  const out: Tree = {};
  for (const k of Object.keys(t).sort()) {
    const v = t[k];
    out[k] = typeof v === "string" ? v : sortTree(v);
  }
  return out;
}

const zhTree: Tree = {};
const sources: Record<string, { text: string; kind: Kind; placeholders: string[]; occurrences: string[] }> = {};

for (const msg of messages.values()) {
  setNested(zhTree, msg.ns, msg.key, msg.text);
  sources[`${msg.ns}.${msg.key}`] = {
    text: msg.text,
    kind: msg.kind,
    placeholders: msg.placeholders,
    occurrences: msg.occurrences.sort(),
  };
}

const sortedSources: typeof sources = {};
for (const k of Object.keys(sources).sort()) sortedSources[k] = sources[k];

fs.mkdirSync(MESSAGES_DIR, { recursive: true });
const write = (name: string, data: unknown) =>
  fs.writeFileSync(path.join(MESSAGES_DIR, name), `${JSON.stringify(data, null, 2)}\n`, "utf8");

/** 读取已有的消息文件。不存在或损坏时返回空树。 */
function readTree(name: string): Tree {
  try {
    return JSON.parse(fs.readFileSync(path.join(MESSAGES_DIR, name), "utf8")) as Tree;
  } catch {
    return {};
  }
}

/**
 * 非破坏性合并。抽取新建的树(fresh)里，顶层命名空间以代码为准，直接采用；
 * 只存在于已有文件(existing)中的顶层命名空间(=手工维护的人工整理命名空间)则保留。
 * 抽取命名空间里过时的(上游已删除的)键会自然消失(便于对照上游)。
 */
function mergeCurated(fresh: Tree, existing: Tree): Tree {
  const extractedTop = new Set(Object.keys(fresh));
  const out: Tree = { ...fresh };
  for (const [ns, subtree] of Object.entries(existing)) {
    if (!extractedTop.has(ns)) out[ns] = subtree;
  }
  return out;
}

write("zh.json", sortTree(mergeCurated(zhTree, readTree("zh.json"))));
write("zh.sources.json", sortedSources);

// --- 汇总报告 ------------------------------------------------------------
const byKind: Record<Kind, number> = { string: 0, template: 0, jsx: 0 };
const byNs = new Map<string, number>();
let totalOccurrences = 0;
for (const msg of messages.values()) {
  byKind[msg.kind]++;
  byNs.set(msg.ns, (byNs.get(msg.ns) ?? 0) + 1);
  totalOccurrences += msg.occurrences.length;
}

// 覆盖率估算：含汉字的源码行中，命中抽取节点的行所占比例。
// 其余几乎都是注释(=非抽取对象)。
let hanLines = 0;
let capturedHanLines = 0;
for (const absFile of scannedFiles) {
  const relFile = path.relative(REPO_WEB_DIR, absFile);
  const lines = fs.readFileSync(absFile, "utf8").split(/\r?\n/);
  const capSet = capturedLines.get(relFile) ?? new Set<number>();
  lines.forEach((ln, i) => {
    if (HAN.test(ln)) {
      hanLines++;
      if (capSet.has(i + 1)) capturedHanLines++;
    }
  });
}

const topNs = [...byNs.entries()].sort((a, b) => b[1] - a[1]).slice(0, 15);

console.log("─".repeat(64));
console.log("ARTEX i18n 抽取完成");
console.log("─".repeat(64));
console.log(`扫描文件数            : ${scannedFiles.length} (src/ 下的 .ts/.tsx，已排除 scripts)`);
console.log(`含汉字的文件          : ${fileWithHan}`);
console.log(`去重后的消息(键)数    : ${messages.size}`);
console.log(`  ├─ 字符串字面量      : ${byKind.string}`);
console.log(`  ├─ 模板字面量        : ${byKind.template}`);
console.log(`  └─ JSX 文本          : ${byKind.jsx}`);
console.log(`出现位置总数          : ${totalOccurrences} (含重复使用)`);
console.log(`命名空间数            : ${byNs.size}`);
console.log("");
console.log(`含汉字的源码行        : ${hanLines}`);
console.log(`  └─ 被抽取捕获的行    : ${capturedHanLines} (${((capturedHanLines / hanLines) * 100).toFixed(1)}%)`);
console.log(`     其余 ${hanLines - capturedHanLines} 行大多是代码注释(非抽取对象)`);
console.log("");
console.log("消息数最多的前 15 个命名空间:");
for (const [ns, n] of topNs) console.log(`  ${String(n).padStart(4)}  ${ns}`);
console.log("");
console.log("输出: web/messages/{zh.json, zh.sources.json}");
console.log("─".repeat(64));
