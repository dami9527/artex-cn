# ARTEX 主机分类（triage）

中文 · [English](README.en.md)

[`artex_host_triage.py`](artex_host_triage.py) 是**在单台可疑主机上直接**运行的只读分类（triage）脚本，用于判断该主机
是否存在 ARTEX 运行过的迹象。本目录其余资料面向运行 SIEM（[Sigma](../sigma/)）·网络传感器
（[Suricata](../suricata/)）·威胁情报平台（[入侵指标](../indicators/)）的防御者。本脚本面向另一类响应人员，也就是
没有 SIEM、站在可疑主机 shell 前，只能凭本地状态快速而有依据地回答「这里跑过 ARTEX 吗」的人。

本脚本逐项检查目录中其它资料所载的指纹，并额外检查 **[入侵指标
列表](../indicators/artex_indicators.csv) 有意不配 Sigma 规则的三个主机·数据库指标**。这三项指标无法从日志或网络观测，
只能在主机上直接确认（`server-listen-port`、`recording-proxy-endpoint`、`postgres-exploration-schema`）。

## 检查内容

所有检查项都依据本仓库源码中核实过的字符串或路径，每项发现都附上与对应 Sigma 规则或入侵指标行相同的局限说明。

- **监听端口**：检查 `:8787`（管理 UI）与 `127.0.0.1:8788`（记录代理）。这两个值是
  [`cmd/artex/main.go`](../../cmd/artex/main.go) 中 `--addr`·`--proxy` 标志的默认值。在运行中的主机上解析
  `ss`·`netstat`·`lsof` 的输出，也可以从 `--ports-from` 传入的文件中读取。
- **记录代理产物**：检查记录器首次运行时生成的中间人（MITM）CA 文件
  `<数据目录>/traffic/_ca/mitmproxy-ca-cert.pem`，以及旁边的 `_index/index.sqlite`·`_blobs/`
  （[`traffic/traffic.go`](../../traffic/traffic.go)；数据目录默认为可执行文件旁的 `data/`）。该 CA 是解密并记录
  往返 HTTP(S) 的中间人流量记录器的信任锚（MITRE ATT&CK T1557）。
- **日志标记**：在日志文件中查找补全探测器的 User-Agent `artex-enrich/1.0`
  （[`enrich/enrich.go`](../../enrich/enrich.go)）、自更新外发流量的 User-Agent `artex-selfupdate`
  （[`selfupdate/github.go`](../../selfupdate/github.go)）以及平台守卫审计标记
  （[`guard/guard.go`](../../guard/guard.go)）。守卫标记连非 ASCII 框架字节都原样保留，使 grep 能真正匹配。经日志
  轮转压缩为 `.gz`·`.bz2`·`.xz` 的历史日志也会解压后一并检查，因此覆盖主机的日志历史。不过，Python 标准库中没有
  对应编解码器的格式（`.zst`·`.lz4`）不会检查，而是**作为已跳过的文件上报**。它不会被悄悄当作干净处理，这类文件
  请先解压或手动 `grep` 单独确认。
- **PostgreSQL 探查模式**：检查 ARTEX 存储中的双图（dual-graph）表（`exploration_nodes`·`_edges`·`_anchors` 与
  `assets`·`companies`·`activity`，以及 `agent_prompts` 种子）（[`db/schema.sql`](../../db/schema.sql)）。给出 DSN 时用
  `psql` 查询；没有 `psql` 时，直接输出可手动执行的只读查询。
- **运行中进程的环境变量注入**：查找**同时**带有代理变量（`HTTP_PROXY`·`HTTPS_PROXY`·`ALL_PROXY`）与指向
  mitmproxy CA（`mitmproxy-ca-cert.pem`）的工具链 CA 信任变量（`SSL_CERT_FILE`·`CURL_CA_BUNDLE`·
  `REQUESTS_CA_BUNDLE`·`GIT_SSL_CAINFO`·`NODE_EXTRA_CA_CERTS`）的进程。ARTEX 会向它生成的每个 worker 工具注入
  的正是这组变量（[`agent/worker.go`](../../agent/worker.go) 的 `proxyEnv`，由
  [`agent/proxyenv_test.go`](../../agent/proxyenv_test.go) 断言）。由于**变量名硬编码在源码中**（只能改值），即使
  操作者改了二进制名称或端口，这个指纹依然存在，比单看监听端口更具特异性。在运行中的 Linux 主机上读取 `/proc`，
  在离线·取证镜像中读取用 `--proc-from` 捕获的环境变量转储。代理与 CA 同时出现时报高严重度；只有 mitmproxy CA，
  或只有 ARTEX 默认代理端点（`127.0.0.1:8788`）时报中严重度；没有 mitmproxy CA 的公司代理不会作为线索上报。

命中是**分类线索而非定性结论**。同时，任何一项都没有命中也不代表安全，因为操作者可以改二进制名称、搬走数据
目录或更换端口。

## 平台支持

本脚本是纯 Python 3（仅标准库），因此只要有 Python 3 就能运行，已在 Linux（CI 自测）与 macOS 上实际运行验证。
依赖操作系统的检查有两项，二者都不会失败，而是干净地降级。

- **运行中端口检查**：依次尝试 `ss` → `netstat` → `lsof`，使用第一个有输出的工具。在 Linux 上使用
  `ss`·`netstat`；在没有 `ss` 且 `netstat` 不接受 Linux 式 `-ltnp` 标志的 macOS·BSD 上（这种情况下它无输出
  退出），会改用 `lsof -nP -iTCP -sTCP:LISTEN`，并以同样方式解析。若不想实时扫描而想读取已保存的清单，请使用
  `--ports-from`。
- **运行中进程环境变量检查**：读取 `/proc`，因此只在 Linux 上有效。在没有 `/proc` 的主机（macOS·BSD）上，
  上报的是**已跳过**而不是干净，请在 Linux 主机上生成转储后用 `--proc-from` 传入（参见使用方法）。

其余检查（记录代理产物·日志标记·PostgreSQL 模式）读取文件系统、日志文件以及（有 DSN 时）`psql`，因此与
操作系统无关。

## 使用方法

```sh
# 对主机做端到端检查
detections/triage/artex_host_triage.py \
    --data-dir /opt/artex/data \
    --log /var/log/syslog --log-dir /var/log/artex \
    --pg-dsn "$ARTEX_PG_DSN"

# 输出机器可读格式，任一项命中即以非 0 退出
detections/triage/artex_host_triage.py --data-dir /opt/artex/data --json --exit-code

# 离线·取证镜像：读取捕获的进程环境变量转储
#   在主机上生成转储的方法：
#   for p in /proc/[0-9]*; do echo "# $p"; tr '\0' '\n' < "$p/environ"; echo; done > proc_env_dump.txt
detections/triage/artex_host_triage.py --proc-from proc_env_dump.txt

# 可复现的夹具自测（不触碰主机状态）
detections/triage/artex_host_triage.py --self-test
```

本脚本只使用纯 Python 3 标准库：无需安装、不使用网络，除 `--self-test` 自己的临时目录外不向任何位置写入。
它读取主机状态（监听端口、数据目录、日志文件，以及仅在给出 DSN 时的数据库）并输出发现的内容。退出码默认为
`0`（用于分类而非门禁）；加上 `--exit-code` 后，只要有指标命中就以 `1` 退出。

## 如何保持如实

`--self-test` 会构造一台合成主机：带有植入的 CA·索引·blob 存储的数据目录、含各标记的日志、端口清单，以及
捕获的进程环境变量转储；随后断言每个检查都能在其上触发，接着断言在干净主机、正常日志与公司代理进程上发现数
为 **0**（没有误报）。该自测已接入
[`detections` CI 工作流](../../.github/workflows/detections.yml)，并由
[`detections/tests/run-all.sh`](../tests/run-all.sh) 再次运行。因此任何检查被改坏，或指标与其 grep 的源码字符串
出现偏差，都会在合并门禁上失败。无法运行的检测只是一句主张。
