# ARTEX 检测规则（Sigma / 主机·日志·SIEM）

中文 · [English](README.en.md)

本目录在 ARTEX 检测套件中负责主机·日志·SIEM 层。这里的
[Sigma](https://sigmahq.io) 规则把防御指南（[中文](../../docs/defense-zh.md) ·
[English](../../docs/defense-en.md)）第 4 节的伪规则正式化为厂商中立的格式，再转换成各自的
SIEM·EDR 查询语言使用。所有指标都依据本仓库源码中实际核实过的字符串或行为，而不是推测。网络层位于
[`../suricata/`](../suricata/)，包含 ATT&CK 图层、
指标 CSV·MISP 导出、主机分类脚本在内的整个检测套件由
[`../README.md`](../README.md) 索引。所有规则仅用于**防御与检测目的**，保护自己拥有或获得书面授权的
系统。

## 原子（atomic）规则

一条规则对应一个可观测的事实。可以单独转换，也可以作为整棵树的一部分转换。

- **[`artex_enrich_user_agent.yml`](artex_enrich_user_agent.yml)**：*ARTEX Asset Enrichment Probe
  User-Agent*。资产补全（`enrich/enrich.go`）发出的入站 `artex-enrich/1.0` User-Agent。
  属于目标侧观测的辅助指标。`level: high`。
- **[`artex_selfupdate_egress.yml`](artex_selfupdate_egress.yml)**：*ARTEX Self-Update Egress
  User-Agent*。自更新例程（`selfupdate/github.go`）发出的出站 `artex-selfupdate`
  User-Agent。属于主机·取证的外发（egress）指标。`level: medium`。
- **[`artex_guard_audit_framing.yml`](artex_guard_audit_framing.yml)**：*ARTEX Platform Guard
  Audit-Log Framing*。工具调用被拦截时写入审计日志的平台守卫控制标记
  （`guard/guard.go`）。主机·取证指标。`level: high`。
- **[`artex_recording_proxy_ca.yml`](artex_recording_proxy_ca.yml)**：*ARTEX Recording-Proxy MITM CA
  Certificate Artifact*。记录代理以 `_ca/mitmproxy-ca-cert.pem` 布局生成的 MITM CA 文件
  （`traffic/traffic.go`）。主机·取证产物，文件名本身与单独运行的 mitmproxy 共用，因此
  按狩猎线索（hunting lead）处理。`level: medium`。
- **[`destructive_command_hunting.yml`](destructive_command_hunting.yml)**：*Destructive Command
  Execution (ARTEX Guard-List Hunting)*。对应 ARTEX 守卫内置拒绝列表（`db/db.go` 种子）的破坏性
  shell·数据库命令。**不是** ARTEX 独有签名，而是通用狩猎线索。`level: medium`。

## 关联（correlation）规则（基于行为）· [`correlation/`](correlation/)

静态字符串可以改，行为更难隐藏。这些 Sigma **关联**规则把防御指南
4.1~4.2 节与 4.4 节的行为层正式化。每条规则都用 `id` 引用上面的一条原子规则，因此必须
**转换整棵 `sigma/` 树**而不是单个关联文件，引用才能解析（[Sigma 测试](../tests/sigma/) 断言的
正是这个依赖关系）。

- **[`correlation/artex_enrich_scan_velocity.yml`](correlation/artex_enrich_scan_velocity.yml)**：
  *Enrichment Scan Velocity*。一个来源在短时间窗口内倾泻 `artex-enrich/1.0` 探测
  （补全以并发 4 运行，没有速率限制）。它抓住单条规则漏掉的速度。`event_count`、
  `level: high`。
- **[`correlation/artex_enrich_fanout.yml`](correlation/artex_enrich_fanout.yml)**：*Enrichment
  Fan-Out*。一个来源把补全 User-Agent 带到多个**互不相同**的主机。信号不是请求量，而是
  接触到的不同主机数量，抓住的是以机器速度扫过资产列表的广度。`value_count`、
  `level: high`。
- **[`correlation/artex_guard_block_burst.yml`](correlation/artex_guard_block_burst.yml)**：
  *Guard-Block Burst*。一台主机上平台守卫控制标记反复出现。它指向的是
  实际运行中的 ARTEX 触碰到自身守卫，而不是仅仅引用了该标记的文档。`event_count`、
  `level: high`。
- **[`correlation/artex_guard_marker_then_destructive.yml`](correlation/artex_guard_marker_then_destructive.yml)**：
  *Guard Marker With Destructive Command*。守卫标记与破坏性命令在同一主机、同一窗口内一起
  出现（防御指南 4.2 节，多阶段）。它把 ARTEX 独有标记与本来通用的破坏性命令信号
  结合，从而提高了特异性。`temporal`、`level: high`。

阈值与窗口是保守的默认值，请按自己的基线调整。纯 Web 多阶段场景（枚举 →
探测 → 认证）仍然需要按环境定制的基础规则，因为该模式无法归约为单个 ARTEX 独有
User-Agent。可作为起点的通用行为基础模板在[防御指南 4.2 节](../../docs/defense-zh.md)
中，由于无法以 ARTEX 源码为依据，有意不放进这棵经过验证的树。

## 范围与如实说明：部署前请阅读

- **静态指标可以改。** 操作者可以换 User-Agent 或删掉 CA 文件，因此没有原子
  指标并不**不**意味着安全。持久的信号是 `correlation/` 规则所依据的
  行为：一个来源从侦察串到枚举、探测、认证·注入尝试，随响应调整，并且不停歇地
  运行。
- **破坏性命令规则属于通用狩猎。** 它对应 ARTEX 守卫的拒绝列表，但同样的命令
  合法管理员也会执行。命中只当线索，用自己的环境做允许列表过滤，不要仅凭它
  就归因于 ARTEX。
- **端口与模式属于主机取证，不是 Sigma。** 服务器默认端口 `:8787` 与记录代理
  `127.0.0.1:8788`（`cmd/artex/main.go`），以及 PostgreSQL 探索图模式，最好在可疑主机上直接
  确认。因此它们不以嘈杂规则的形式提供，而是放在[指标 CSV](../indicators/)与
  [主机分类脚本](../triage/)里。
- **`logsource` 与字段名是通用值。** 规则使用通用的 `category`·`product` 日志源与字段名
  （`cs-user-agent`、`CommandLine`、`TargetFilename`）。转换时请用流水线（`-p`）把它们映射到自己
  产品的模式；参见下面的后端说明。

## 验证与转换

已用 [sigma-cli](https://github.com/SigmaHQ/sigma-cli)（pySigma）验证。在仓库根目录执行。

```sh
python3 -m venv .venv && . .venv/bin/activate
pip install sigma-cli

# 结构 + 最佳实践验证（预期：0 errors, 0 issues）
sigma check detections/sigma/

# 以该规则集记录在案的基线检查完整 SigmaHQ 约定（预期：0 issues）
pip install pySigma-validators-sigmahq
sigma check --validation-config detections/tests/sigma_lint/validators.yml detections/sigma/

# 转换整棵树，使关联规则能解析它按 id 引用的原子规则
sigma plugin install splunk
sigma convert -t splunk --without-pipeline detections/sigma/
```

各后端对关联规则的支持不同，因此 `-t` 的选择很重要。Splunk、Elasticsearch EQL、Grafana Loki 转换
整棵树，而 Elasticsearch Lucene、OpenSearch 与微软 `kusto` 后端只转换五条原子
规则（窗口在产品中原生表达）。按后端实测的表格与 `--without-pipeline`·
`-p` 字段映射说明在 [`../README.md`](../README.md) 中，
并由 [`../tests/sigma_backends/`](../tests/sigma_backends/) 复现。

## 测试

[`../tests/`](../tests/) 下有四个可复现的套件覆盖这些规则，每个都只需要 Docker。
[`sigma/`](../tests/sigma/) 断言验证与整树编译，以及关联规则单独转换会失败，
[`sigma_match/`](../tests/sigma_match/) 确认规则在恶意样本上确实触发、在良性样本上
保持沉默，[`sigma_backends/`](../tests/sigma_backends/) 检查五种后端的移植性，
[`sigma_lint/`](../tests/sigma_lint/) 检查完整 SigmaHQ 验证器基线（0 issues）。
参见 [`../tests/README.md`](../tests/README.md)。

## 贡献

欢迎贡献检测规则。新规则应让每个指标都依据可观测的事实，在 `description` 中
写明局限，干净地通过 SigmaHQ 验证器基线
（`sigma check --validation-config ../tests/sigma_lint/validators.yml .`），并且
不含任何读起来像攻击指引的内容。参见 [`../../CONTRIBUTING.md`](../../CONTRIBUTING.md)、
[`../suricata/`](../suricata/) 的网络层，以及 [`../README.md`](../README.md)。
