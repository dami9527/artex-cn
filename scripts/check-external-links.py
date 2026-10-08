#!/usr/bin/env python3
"""检查被跟踪的 Markdown 文档所引用的外部链接（http、https）是否仍然可用。

姊妹脚本 `check-doc-links.py` 只看仓库内部的链接与锚点，外部 URL 按设计跳过
（依赖网络会不稳定，不放进合并门禁）。那份「另行检查」就由本脚本承担。README
顶部、防御指南、detections README 向访客与防御者推荐的外部链接（事件报告入口
CNCERT/CC、12377 举报中心、CISA KEV，OWASP、SigmaHQ、Suricata、MITRE、MISP，
原始演示 artex-demo.vercel.app，以及 GitHub 徽章等）一旦失效、迁移或关闭，就会
在无人察觉的情况下一直烂在那里；本脚本定期、手动地把它们捞出来。

挑选待检查 URL 的规则
- 遍历所有被跟踪的 `.md`，但跳过围栏代码块（``` 或 ~~~）与行内代码跨度
  （`` `...` ``）里的 URL。那些 URL 属于命令示例、配置值或引用的外部资产
  （例如某份验证文档里的 `id.redhaze.top`），并不是「读者会去点的参考链接」。
- 从 Markdown 链接与图片（`[text](url)`、`![alt](url)`）、自动链接（`<url>`），
  以及剩余正文中的裸 URL 里收集 http、https 地址。
- 排除保留与占位主机：localhost、私有与回环 IP（127.、10.、192.168.、169.254.、
  172.16~31.、0.0.0.0、::1）、RFC 2606/6761 保留域（example.com/org/net/edu、
  `*.example.*`、`.test`、`.invalid`、`.local`、`.tld`），以及不含点、不是 FQDN 的
  名字（例如 `target`）。

如何确认可用（把 MAINTAINING.md 8.2 的经验写进代码）
公共与安全机构的站点常常拒绝 HEAD 请求与默认 User-Agent，或者连续多次重定向，
因此简单探测会把好好的链接误判为断裂。所以本脚本用**浏览器 User-Agent、GET 方式、
跟随重定向**来确认，并把结果分成三类。
- OK：最终状态是 2xx、3xx，链接有效。
- RESTRICTED：401、403、405、429。主机是活的，只是确认方式被服务端策略拦住了
  （反爬、拒绝该方法、限速），并不是坏链接。会报告，但不计为失败。
- DOWN：404、410、5xx（重试后依然）、DNS 或连接或超时或 SSL 错误（重试后依然）。
  很可能是真的断了。

网络错误、5xx、429 会以小幅延迟重试，用来区分偶发抖动和真正的故障。
只用标准库。本脚本只对仓库**公开文档已经引用**的 URL 发 GET 确认存活，
不扫描、不探测任何目标。

已知例外（allowlist）
`scripts/external-links-allowlist.txt` 里列出的 URL 若判为 DOWN，会单独归类为
"ALLOWED"，且不计入 strict 退出码。这是为了把「我们不拥有、也修不了」的上游原文
保存文件（例如上游 CHANGELOG.zh.md）继承下来的死链透明地记录下来，让周期性 strict
检查不会因为它们而永久飘红，而**只在新出现坏链时才变红**。

执行
- 默认（只报告，退出码 0）：`python3 -I scripts/check-external-links.py`
- 周期性或发布前检查（有新坏链就变红，不用于合并门禁）：`--strict`
  （只要有一个不在 allowlist 里的 DOWN 就以退出码 1 结束）。
  RESTRICTED 与 ALLOWED 在 strict 下也不算失败。
- 不联网、只看抽取出的集合：`--list`（不发起请求，只打印待检查 URL 与出处）。
"""
import argparse
import http.client
import os
import re
import socket
import ssl
import subprocess
import sys
import time
import urllib.error
import urllib.request
from collections import defaultdict
from urllib.parse import urlsplit

# [text](url) 正文链接（非图片）与 ![alt](url) 图片。路径截到空格或右括号前。
MD_LINK = re.compile(r"(?<!\!)\[[^\]]*\]\(([^)\s]+)")
MD_IMAGE = re.compile(r"!\[[^\]]*\]\(([^)\s]+)")
# <https://...> 自动链接。
AUTOLINK = re.compile(r"<(https?://[^>\s]+)>")
# 正文里的裸 URL。尾随标点稍后剥掉。
BARE_URL = re.compile(r"https?://[^\s)>\]\"'`]+")
# 行内代码跨度 `...`（单反引号）。抽取前先替换成空格。
INLINE_CODE = re.compile(r"`[^`]*`")
# 正文 URL 结尾常见的标点。
TRAILING_PUNCT = ".,;:!?\"'»)]}>"

# IPv4 私有、回环、链路本地网段（172.16~31. 单独一个模式）。
PRIVATE_IPV4 = re.compile(r"^(127\.|10\.|192\.168\.|169\.254\.|0\.0\.0\.0$)")
PRIVATE_IPV4_172 = re.compile(r"^172\.(1[6-9]|2\d|3[01])\.")

BROWSER_UA = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
    "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0 Safari/537.36"
)

# 主机是活的，只是确认方式被服务端策略挡住的状态码（不是断裂）。
RESTRICTED_CODES = {401, 403, 405, 429}

ALLOWLIST_PATH = os.path.join("scripts", "external-links-allowlist.txt")


def load_allowlist(root: str) -> set:
    """读取我们修不了的已知 DOWN URL 集合（文件不存在则为空集）。

    每行一个 URL。`#` 之后是注释，空行与纯注释行忽略。
    """
    path = os.path.join(root, ALLOWLIST_PATH)
    allow = set()
    if not os.path.exists(path):
        return allow
    with open(path, encoding="utf-8") as fh:
        for line in fh:
            line = line.split("#", 1)[0].strip()
            if line:
                allow.add(line)
    return allow


def strip_code(line: str) -> str:
    """去掉围栏之外某一行里的行内代码跨度（其中的 URL 不算引用）。"""
    return INLINE_CODE.sub(" ", line)


def is_checkable(url: str) -> bool:
    """过滤掉保留与占位主机，只留下真正值得检查的外部 URL。"""
    parts = urlsplit(url)
    if parts.scheme not in ("http", "https"):
        return False
    host = parts.hostname
    if not host:
        return False
    host = host.lower()
    if host == "localhost" or host.endswith(".localhost"):
        return False
    if host == "::1":
        return False
    if PRIVATE_IPV4.match(host) or PRIVATE_IPV4_172.match(host):
        return False
    if host in ("example.com", "example.org", "example.net", "example.edu"):
        return False
    if host.endswith((".example.com", ".example.org", ".example.net", ".example")):
        return False
    if host.endswith((".test", ".invalid", ".local", ".localdomain", ".tld")):
        return False
    if "." not in host:  # 不含点的裸名字（例如 target），不是 FQDN
        return False
    return True


def extract(root: str, md_files: list) -> dict:
    """从被跟踪的 .md 里构造「待检查外部 URL → [(文件, 行号), ...]」字典。"""
    found = defaultdict(list)
    for md in md_files:
        path = os.path.join(root, md)
        # 索引与工作区可能暂时不一致（文件刚被重命名、还没暂存），这种情况跳过。
        if not os.path.exists(path):
            continue
        with open(path, encoding="utf-8", errors="ignore") as fh:
            lines = fh.readlines()
        in_fence = False
        for lineno, raw in enumerate(lines, 1):
            stripped = raw.lstrip()
            if stripped.startswith("```") or stripped.startswith("~~~"):
                in_fence = not in_fence
                continue
            if in_fence:
                continue
            line = strip_code(raw)
            candidates = []
            for pattern in (MD_LINK, MD_IMAGE, AUTOLINK):
                candidates.extend(m.group(1) for m in pattern.finditer(line))
            for m in BARE_URL.finditer(line):
                candidates.append(m.group(0).rstrip(TRAILING_PUNCT))
            for url in candidates:
                url = url.strip().rstrip(TRAILING_PUNCT)
                if is_checkable(url):
                    where = (md, lineno)
                    if where not in found[url]:
                        found[url].append(where)
    return found


def probe(url: str, timeout: float, retries: int, delay: float):
    """用浏览器 UA 对 URL 发 GET，返回 (分类, 详情)。

    分类为 "OK"、"RESTRICTED"、"DOWN"。网络错误、5xx、429 会重试。
    """
    req = urllib.request.Request(
        url,
        method="GET",
        headers={
            "User-Agent": BROWSER_UA,
            "Accept": "*/*",
            "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
        },
    )
    last = ""
    for attempt in range(retries + 1):
        try:
            with urllib.request.urlopen(req, timeout=timeout) as resp:
                code = resp.getcode()
                if code in RESTRICTED_CODES:
                    return "RESTRICTED", f"HTTP {code}"
                return "OK", f"HTTP {code}"
        except urllib.error.HTTPError as exc:
            code = exc.code
            if code in RESTRICTED_CODES:
                return "RESTRICTED", f"HTTP {code}"
            if code >= 500 or code == 408:
                last = f"HTTP {code}"  # 可能是暂时的，重试
            else:
                return "DOWN", f"HTTP {code}"  # 404、410 等已经确定，不必重试
        except (
            urllib.error.URLError,
            http.client.HTTPException,
            ssl.SSLError,
            socket.timeout,
            ConnectionError,
            TimeoutError,
            OSError,
        ) as exc:
            reason = getattr(exc, "reason", exc)
            last = f"{type(exc).__name__}: {reason}"
        if attempt < retries:
            time.sleep(delay * (attempt + 1))
    return "DOWN", last


def main() -> int:
    ap = argparse.ArgumentParser(description="检查被跟踪 Markdown 里的外部链接是否存活")
    ap.add_argument("--strict", action="store_true",
                    help="只要有 DOWN 就以退出码 1 结束（用于周期性或发布前检查）")
    ap.add_argument("--list", action="store_true",
                    help="不发起网络请求，只打印待检查的 URL 与出处")
    ap.add_argument("--timeout", type=float, default=15.0, help="请求超时（秒）")
    ap.add_argument("--retries", type=int, default=2, help="网络错误、5xx、429 的重试次数")
    ap.add_argument("--delay", type=float, default=0.5, help="调用之间与重试之间的基础延迟（秒）")
    args = ap.parse_args()

    root = subprocess.check_output(
        ["git", "rev-parse", "--show-toplevel"], text=True
    ).strip()
    tracked = subprocess.check_output(["git", "ls-files"], cwd=root, text=True).splitlines()
    md_files = [f for f in tracked if f.endswith(".md")]

    allow = load_allowlist(root)
    found = extract(root, md_files)
    urls = sorted(found)
    print(f"从 {len(md_files)} 篇被跟踪的 Markdown 中收集到待检查外部 URL {len(urls)} 个")

    if args.list:
        for url in urls:
            where = ", ".join(f"{f}:{ln}" for f, ln in found[url])
            tag = " [allowlist]" if url in allow else ""
            print(f"  {url}{tag}  ({where})")
        return 0

    results = {"OK": [], "RESTRICTED": [], "ALLOWED": [], "DOWN": []}
    for i, url in enumerate(urls):
        if i:
            time.sleep(args.delay)  # 调用之间稍作延迟（避免触发沙箱调节）
        verdict, detail = probe(url, args.timeout, args.retries, args.delay)
        if verdict == "DOWN" and url in allow:
            verdict = "ALLOWED"  # 修不了的、已知从上游继承下来的 DOWN
        results[verdict].append((url, detail))
        print(f"  [{verdict:10}] {url} — {detail}")

    print(
        f"\n汇总：OK {len(results['OK'])} · RESTRICTED {len(results['RESTRICTED'])} · "
        f"ALLOWED {len(results['ALLOWED'])} · DOWN {len(results['DOWN'])}"
    )
    if results["RESTRICTED"]:
        print("RESTRICTED（主机存活、确认方式被拦，不是坏链接）：")
        for url, detail in results["RESTRICTED"]:
            print(f"  {url} — {detail}")
    if results["ALLOWED"]:
        print("ALLOWED（allowlist 里已知的 DOWN，我们修不了，strict 不计算）：")
        for url, detail in results["ALLOWED"]:
            where = ", ".join(f"{f}:{ln}" for f, ln in found[url])
            print(f"  {url} — {detail}  (引用：{where})")
    if results["DOWN"]:
        print("DOWN（很可能是断了，需要确认）：")
        for url, detail in results["DOWN"]:
            where = ", ".join(f"{f}:{ln}" for f, ln in found[url])
            print(f"  {url} — {detail}  (引用：{where})")

    if args.strict and results["DOWN"]:
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
