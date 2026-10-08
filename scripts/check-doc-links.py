#!/usr/bin/env python3
"""检查仓库内 Markdown 文档中内部链接、图片与锚点引用的完整性。

遍历所有被跟踪的 `.md` 文档，核对两件事。

1. 指向仓库内其他文件的相对路径链接（`[text](path)`）与图片引用
   （`![alt](path)`、`<img src="path">`）是否指向真实存在的文件。
2. 文档锚点链接（`[text](#heading)`、`[text](other.md#heading)`）是否指向目标
   `.md` 文档里确实存在的标题。锚点 slug 用标准库复现 GitHub 的规则
   （转小写 → 只保留 `\\p{Word}`、连字符与空格，其余丢弃 → 空格换成连字符，
   同一 slug 重复出现时按出现顺序加 `-1`、`-2` 后缀）生成。

只要有任意一处引用断裂，就以退出码 1 结束，因此可以直接用于 CI 合并门禁
（`.github/workflows/docs.yml`）和贡献者的本地校验。

不在检查范围内的情况：
- 外部 URL（`http://`、`https://`、`mailto:`、`tel:`、`data:`）：依赖网络会不稳定，
  这个确定性门禁不处理，外部链接状态另行检查。
- 指向非 `.md` 文件的链接里 `#` 之后的部分（例如源码行号）：那不是标题锚点，
  所以只校验文件是否存在，不校验锚点。
- 图片引用里的 `#` 片段：图片没有标题锚点的概念。
- 围栏代码块（用 ``` 或 ~~~ 包裹的块）里的示例链接与形似标题的注释：
  它们是代码示例而不是真实引用或标题，直接跳过。

只用标准库，不访问网络。执行：`python3 -I scripts/check-doc-links.py`
"""
import os
import re
import subprocess
import sys
import unicodedata
from collections import defaultdict

# [text](path) 中非图片（`![...]`）的链接。路径截到空格前（排除标题 "..."）。
MD_LINK = re.compile(r"(?<!\!)\[[^\]]*\]\(([^)\s]+)")
# ![alt](path) 图片链接。
MD_IMAGE = re.compile(r"!\[[^\]]*\]\(([^)\s]+)")
# <img ... src="path" ...> 形式的 HTML 图片。
HTML_IMAGE = re.compile(r"<img[^>]*\bsrc=[\"']([^\"']+)[\"']", re.IGNORECASE)

# 网络与非文件协议。锚点（`#`）指同一文档，不放进这里。
EXTERNAL_PREFIXES = ("http://", "https://", "mailto:", "tel:", "data:")

ATX_HEADING = re.compile(r"^(#{1,6})\s+(.*?)\s*#*\s*$")
# 去掉标题文本里的行内 Markdown，只留下纯文本。
MD_INLINE_LINK = re.compile(r"\[([^\]]*)\]\([^)]*\)")


def is_external(target: str) -> bool:
    return target.startswith(EXTERNAL_PREFIXES)


def _is_word_char(ch: str) -> bool:
    """判断该字符是否为 GitHub 锚点 slug 规则会保留的 `\\p{Word}` 字符。

    Ruby 正则 `\\p{Word}` = Letter(L*) + Mark(M*) + Decimal_Number(Nd)
    + Connector_Punctuation(Pc) + Join_Control(U+200C、U+200D)。汉字、下划线、
    以及 emoji 的 variation selector(Mn) 都会被保留。
    """
    if ch == "_":
        return True
    if ch in ("‌", "‍"):
        return True
    category = unicodedata.category(ch)
    if category[0] in ("L", "M"):
        return True
    return category in ("Nd", "Pc")


def slugify(text: str) -> str:
    """把标题文本转成 GitHub 的锚点 slug（不含重复后缀的处理）。"""
    text = text.strip().lower()
    out = []
    for ch in text:
        if ch == " ":
            out.append("-")
        elif ch == "-" or _is_word_char(ch):
            out.append(ch)
    return "".join(out)


def heading_slugs(root: str, md: str) -> set:
    """收集文件里所有 ATX 标题的 GitHub 锚点 slug。

    同一 slug 重复时，像 GitHub 那样按出现顺序加 `-1`、`-2` 后缀。
    """
    slugs = set()
    counts = defaultdict(int)
    in_fence = False
    with open(os.path.join(root, md), encoding="utf-8", errors="ignore") as fh:
        for line in fh:
            stripped = line.lstrip()
            if stripped.startswith("```") or stripped.startswith("~~~"):
                in_fence = not in_fence
                continue
            if in_fence:
                continue
            match = ATX_HEADING.match(line.rstrip("\n"))
            if not match:
                continue
            raw = match.group(2)
            raw = MD_INLINE_LINK.sub(r"\1", raw)
            raw = raw.replace("`", "").replace("*", "").replace("_", "")
            base = slugify(raw)
            seen = counts[base]
            counts[base] += 1
            slugs.add(base if seen == 0 else f"{base}-{seen}")
    return slugs


def main() -> int:
    root = subprocess.check_output(
        ["git", "rev-parse", "--show-toplevel"], text=True
    ).strip()
    tracked = subprocess.check_output(
        ["git", "ls-files"], cwd=root, text=True
    ).splitlines()
    md_files = [f for f in tracked if f.endswith(".md")]

    slug_cache: dict = {}

    def slugs_of(md_path: str) -> set:
        if md_path not in slug_cache:
            slug_cache[md_path] = heading_slugs(root, md_path)
        return slug_cache[md_path]

    checked_files = 0
    checked_anchors = 0
    broken_files = []
    broken_anchors = []
    for md in md_files:
        # 索引与工作区可能暂时不一致（例如文件刚被重命名、还没暂存）。
        # 这种情况直接跳过，避免整个检查因为一个不存在的路径而崩掉。
        if not os.path.exists(os.path.join(root, md)):
            continue
        with open(os.path.join(root, md), encoding="utf-8", errors="ignore") as fh:
            lines = fh.readlines()
        base_dir = os.path.dirname(md)
        in_fence = False
        for lineno, line in enumerate(lines, 1):
            stripped = line.lstrip()
            if stripped.startswith("```") or stripped.startswith("~~~"):
                in_fence = not in_fence
                continue
            if in_fence:
                continue
            for pattern in (MD_LINK, MD_IMAGE, HTML_IMAGE):
                for match in pattern.finditer(line):
                    target = match.group(1).strip()
                    if is_external(target):
                        continue

                    if target.startswith("#"):
                        file_part, anchor = "", target[1:]
                    elif "#" in target:
                        file_part, anchor = target.split("#", 1)
                    else:
                        file_part, anchor = target, ""
                    file_part = file_part.split("?", 1)[0]
                    anchor = anchor.strip()

                    # 被指向的 .md 文档（锚点检查的目标）。指向其他文件时用那个文件，
                    # 没有文件部分（纯 `#锚点`）时用当前文档。
                    target_md = None
                    if file_part:
                        checked_files += 1
                        resolved = os.path.normpath(
                            os.path.join(root, base_dir, file_part)
                        )
                        if not os.path.exists(resolved):
                            broken_files.append((md, lineno, target))
                            continue
                        if file_part.endswith(".md"):
                            target_md = os.path.normpath(
                                os.path.join(base_dir, file_part)
                            )
                    else:
                        target_md = md

                    # 锚点只对正文链接（MD_LINK）有意义，图片引用里的 `#` 片段跳过。
                    if anchor and target_md is not None and pattern is MD_LINK:
                        checked_anchors += 1
                        if anchor not in slugs_of(target_md):
                            broken_anchors.append((md, lineno, target, target_md))

    print(
        f"已跟踪 Markdown {len(md_files)} 篇 · 文件引用 {checked_files} 处 · "
        f"锚点引用 {checked_anchors} 处"
    )
    if broken_files:
        print(f"断裂的文件引用 {len(broken_files)} 处 —— 指向的文件不在仓库里：")
        for md, lineno, target in broken_files:
            print(f"  {md}:{lineno} -> {target}")
    if broken_anchors:
        print(f"断裂的锚点引用 {len(broken_anchors)} 处 —— 目标文档里没有该标题：")
        for md, lineno, target, target_md in broken_anchors:
            print(f"  {md}:{lineno} -> {target}  (目标：{target_md})")
    if broken_files or broken_anchors:
        return 1
    print("断裂引用 0 —— 所有内部链接、图片与锚点都指向真实存在的目标。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
