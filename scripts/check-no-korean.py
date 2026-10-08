#!/usr/bin/env python3
"""扫描工作树下的文本文件，确认任何地方都不再有韩文（谚文）残留。

本仓库是 ARTEX 中文版：界面、注释、后端用户可见文案、提示词、文档、脚本与入库
数据一律使用简体中文。历史上这里曾经是韩文版，韩文散落在 Go 源码、TSX 组件、
i18n 文案、Markdown 文档、CI 配置与 shell 脚本里。这个脚本把那条约束固化成
可执行的检查，避免以后合入时又悄悄带回韩文。

判定范围
--------
只查谚文，不查汉字、也不查假名：

- 谚文音节（U+AC00–U+D7A3）
- 谚文字母（U+1100–U+11FF、U+A960–U+A97F）
- 谚文兼容字母（U+3130–U+318F）
- 谚文扩展区 B（U+D7B0–U+D7FF，含中世韩文）

汉字与假名不在范围内：中文文案本来就全是汉字，查汉字等于自我误报。假名属于
日文，不属于本次清理目标。

跳过内容
--------
- 版本控制与依赖目录（.git、node_modules、.next、out、dist）
- 构建产物（server/webui/dist、web/out、dist/）
- 二进制文件（无法按 UTF-8 解码的一律跳过）
- 显式忽略清单 IGNORED（当前为空：仓库里不应该有任何需要豁免的文件）

用法
----
    python3 -I scripts/check-no-korean.py            # 全仓库扫描
    python3 -I scripts/check-no-korean.py --quiet    # 只输出结论

只用标准库，不联网。
"""

from __future__ import annotations

import argparse
import os
import re
import sys

# 谚文码位。汉字、假名故意不在其中，避免与中文文案互相误报。
HANGUL = re.compile(
    "["
    "\uac00-\ud7a3"  # 谚文音节
    "\u1100-\u11ff"  # 谚文字母
    "\u3130-\u318f"  # 谚文兼容字母
    "\ua960-\ua97f"  # 谚文字母扩展 A
    "\ud7b0-\ud7ff"  # 谚文扩展 B
    "]"
)

# 与扫描无关的目录名，出现在路径任意一段即整棵子树跳过。
SKIP_DIRS = {
    ".git",
    ".next",
    ".venv",
    "__pycache__",
    "dist",
    "node_modules",
    "out",
}

# 需要豁免的相对路径（当前为空）。
IGNORED: set[str] = set()


def is_skipped(rel: str) -> bool:
    parts = rel.split(os.sep)
    if any(part in SKIP_DIRS for part in parts):
        return True
    # 构建产物：前端静态导出的副本，内容由 web/ 决定，本身不该单独修。
    return rel.startswith(os.path.join("server", "webui", "dist") + os.sep)


def scan(root: str) -> list[tuple[str, int, str]]:
    """返回 [(相对路径, 行号, 该行内容)]，按路径排序。"""
    hits: list[tuple[str, int, str]] = []
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in SKIP_DIRS]
        for name in filenames:
            abs_path = os.path.join(dirpath, name)
            rel = os.path.relpath(abs_path, root)
            if rel in IGNORED or is_skipped(rel):
                continue
            try:
                with open(abs_path, "rb") as fh:
                    raw = fh.read()
                text = raw.decode("utf-8")
            except (OSError, UnicodeDecodeError):
                # 读不到或不是 UTF-8 文本（二进制资源）→ 不在检查范围。
                continue
            for lineno, line in enumerate(text.splitlines(), 1):
                if HANGUL.search(line):
                    hits.append((rel, lineno, line.strip()))
    hits.sort(key=lambda item: (item[0], item[1]))
    return hits


def main() -> int:
    parser = argparse.ArgumentParser(description="检查仓库里是否还有韩文残留")
    parser.add_argument("--quiet", action="store_true", help="只输出结论")
    parser.add_argument("--root", default=None, help="仓库根目录，默认取脚本上一级")
    args = parser.parse_args()

    root = args.root or os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    hits = scan(root)

    if not hits:
        if not args.quiet:
            print("[+] 未发现韩文残留。")
        return 0

    if not args.quiet:
        print(f"[x] 发现 {len(hits)} 行韩文残留：", file=sys.stderr)
        for rel, lineno, line in hits:
            print(f"    {rel}:{lineno}: {line[:160]}", file=sys.stderr)
    print(
        f"[x] 韩文检查未通过：{len(hits)} 行仍含谚文，请改为简体中文。",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    sys.exit(main())
