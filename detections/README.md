# ARTEX 检测规则

中文 · [English](README.en.md)

> 本目录把[防御与检测指南(../docs/defense-zh.md)](../docs/defense-zh.md)第 4 节「检测规则」中的
> 伪规则，转换成各自 SIEM·EDR 查询语言后即可直接部署的厂商中立
> [Sigma](https://sigmahq.io) 格式。这里的所有指标都**不是推测**，而是依据本
> 仓库源码中实际核实过的字符串或行为。所有规则仅用于**防御与检测目的**，保护自己拥有或获得书面
> 授权的系统。

## 原子（atomic）规则

- **`sigma/artex_enrich_user_agent.yml`**：ARTEX 资产补全（`enrich/enrich.go`）发送的入站
  `artex-enrich/1.0` User-Agent。属于目标侧观测的辅助指标。`level: high`。
- **`sigma/artex_selfupdate_egress.yml`**：自更新例程（`selfupdate/github.go`）发出的
  出站 `artex-selfupdate` User-Agent。属于主机·取证视角的外发（egress）指标。
  `level: medium`。
- **`sigma/artex_guard_audit_framing.yml`**：工具调用被拦截时写入审计日志的平台
  守卫控制标记（`guard/guard.go`）。主机·取证指标。`level: high`。
- **`sigma/destructive_command_hunting.yml`**：原样对应 ARTEX 守卫默认阻止列表（`db/db.go` 种子）的
  破坏性 shell·数据库命令。它不是 ARTEX 独有签名，而是通用狩猎线索。
  `level: medium`。
- **`sigma/artex_recording_proxy_ca.yml`**：记录代理以 `_ca/mitmproxy-ca-cert.pem` 布局
  生成的 MITM CA 证书文件（`traffic/traffic.go`）。主机·取证产物，文件名
  本身与单独运行的 mitmproxy 共用，因此按狩猎线索处理。`level: medium`。

## 关联（correlation）规则：基于行为

静态字符串可以改，行为更难隐藏。[`sigma/correlation/`](sigma/correlation/)
中的 Sigma **关联**规则承载防御指南（4.1~4.2 节、4.4 节）的行为层。每条关联规则都用
`id` 引用上面的原子规则，因此要解析引用，必须转换整棵 `sigma/` 树，而不是单个关联
文件（见下文）。

- **`sigma/correlation/artex_enrich_scan_velocity.yml`**：一个来源在短时间窗口内倾泻的
  `artex-enrich/1.0` 探测（补全以并发 4、无速率限制运行）。它抓住单条规则
  漏掉的速度。`event_count`、`level: high`。
- **`sigma/correlation/artex_enrich_fanout.yml`**：一个来源把补全 User-Agent 带到多个**互不相同**的
  主机。表现为以机器速度铺向整个资产列表，线索不只是量（volume），
  更是宽度（breadth）。`value_count`、`level: high`。
- **`sigma/correlation/artex_guard_block_burst.yml`**：一台主机上平台守卫控制标记反复
  出现。它不是仅仅引用了该标记的文档，而是正在运行的 ARTEX 触碰自身守卫
  的信号。`event_count`、`level: high`。
- **`sigma/correlation/artex_guard_marker_then_destructive.yml`**：同一主机在同一时间窗口内同时出现守卫
  标记与破坏性命令（防御指南 §4.2，多阶段）。它把 ARTEX 独有标记与
  本身通用的破坏命令信号结合，提高了特异性。`temporal`、`level: high`。

阈值与时间窗口是保守的默认值。请按各自的基线（baseline）调整。§4.2 的纯
Web 多阶段场景（枚举 → 探测 → 认证）无法归约为单个 ARTEX 独有 User-Agent，
因此仍然需要按环境定制的基础规则。可作为起点的通用行为 Sigma 基础
模板放在[防御指南 §4.2](../docs/defense-zh.md#42-siem-关联规则)中；由于无法以 ARTEX 源码固定
依据，它没有放进这里经过测试的规则树。

## 网络规则（Suricata）

Sigma 处理主机与日志遥测。在线路上可观测的 ARTEX 独有 User-Agent 有两
个，二者都以 [Suricata](https://suricata.io) 规则放在 [`suricata/`](suricata/) 中。补全
探测器的 `artex-enrich/1.0`（`enrich/enrich.go`）对应一条存在性签名与一条高速枚举变体（sid
1000001·1000002），norma SDK 的 WebFetch 工具在攻击阶段发送的 `norma/0.4`（`github.com/Autumn-27/norma/tool/webfetch.go`）
对应一条存在性签名（sid 1000003）。其它 worker 工具（用 Bash 运行的 `curl`·`nmap` 等）
使用自身 User-Agent，没有 ARTEX 独有指纹，因此网络层有意只狭窄地
覆盖这两个 UA。范围、TLS 注意事项，以及用 `suricata -T` 与参考 pcap 验证的方法见
[`suricata/README.md`](suricata/README.md)。

## ATT&CK 覆盖率

这些规则标记的技术汇总在 [MITRE ATT&CK](https://attack.mitre.org/) Navigator 图层
[`attack/artex_navigator_layer.json`](attack/) 中。它包含跨六个战术（侦察、命令与控制、执行、影响、
凭证访问、收集）的八项技术，每项技术都依据规则的 `attack.*` 标签，并按检测强度（ARTEX 独有
签名，还是通用狩猎线索）打分。在 [ATT&CK Navigator](https://mitre-attack.github.io/attack-navigator/)
中打开，就能看到哪种 ARTEX 行为被哪条规则覆盖。分数计算、技术与规则的对应，以及
如实的范围（覆盖率不是完整性）见 [`attack/README.md`](attack/README.md)。
[一致性测试](tests/attack/run.sh) 会保证图层与规则集彼此不出现偏差。

## 入侵指标列表（机器可读）

为需要原子指标本身而不是检测逻辑的防御者，
[`indicators/artex_indicators.csv`](indicators/) 把 ARTEX 发出的唯一指纹汇总到一个 CSV
文件中，便于直接导入威胁情报平台、SIEM 查询表或主机分类（triage）检查表。它包含补全与自
更新 User-Agent、守卫审计标记、服务器·代理默认端点、记录
代理 CA 证书，以及 PostgreSQL 探索图模式指纹，每行都写明作为依据的源
文件与（如果有）建立在它之上的规则。同样的指标也以可直接
导入的 [MISP](https://www.misp-project.org/) 事件
（[`indicators/artex_indicators.misp.json`](indicators/)）提供，因此使用 MISP 或从它
导出 STIX 的防御者无需手工映射 CSV 列。基于规则的指纹标了 `to_ids`，
主机取证用的端口与模式指纹则没有标。通用狩猎线索（破坏命令）与
norma SDK 共用的 `norma/0.4` WebFetch User-Agent（由 Suricata sid 1000003 捕获的网络签名，
并非 ARTEX 独有字符串）为避开误报，有意不放进可导入列表。列构成、MISP 类型映射、如实的注意事项，以及
保证 CSV 与 MISP 事件不出现偏差的一致性测试见 [`indicators/README.md`](indicators/README.md)。

## 主机分类（triage）

上述规则面向使用 SIEM·网络传感器·威胁情报平台的防御者。针对另一类响应人员，也就是
没有 SIEM、站在单台可疑主机 shell 前的人，本仓库提供 [`triage/artex_host_triage.py`](triage/)。它是
仅凭本地状态回答「这里跑过 ARTEX 吗」的只读脚本。它检查同样的指纹，并额外检查
**CSV 有意不配 Sigma 规则的三个主机·数据库指标**（服务器监听端口、记录代理端点、
PostgreSQL 探查模式）。这三项无法从日志或网络观测，只能在主机上直接确认。
它还检查记录器注入子进程的环境变量痕迹，也就是运行中的进程是否同时带有代理变量与
mitmproxy CA 信任变量（`agent/worker.go`），从 `/proc` 或 `--proc-from` 转储中读取。
每项发现都是与其对应入侵指标行具有同样局限的分类线索。详见
[`triage/README.md`](triage/README.md)，内置的 `--self-test` 会作为下面的合并门禁运行。

## 测试

规则在 [`tests/`](tests/) 中附带了只需 Docker 就能运行的可复现测试。

- **Suricata**（[`tests/suricata/run.sh`](tests/suricata/run.sh)）：先验证整个规则文件能否用
  `suricata -T --init-errors-fatal` 加载（即使没有抓包触发规则，解析失败也会被抓到），再用 scapy
  合成确定性抓包，并用 `suricata -r` 在其上运行，断言存在性规则每个探测触发一次、速度规则在超过
  阈值时命中，并且在良性（benign）User-Agent 抓包上告警为 0。二进制抓包不入库，每次运行都重新生成。
- **Sigma**（[`tests/sigma/run.sh`](tests/sigma/run.sh)）：把下面「验证与转换」中的 `sigma check` 与
  `sigma convert` 验证作为可执行的测试运行。它断言错误为 0、整棵树能编译成后端查询、
  每个原子指标字符串都能存活到该查询中，以及关联规则单独转换会失败。
  最后一条断言证明它确实依赖所引用的原子规则。
- **Sigma 实时事件匹配**（[`tests/sigma_match/run.sh`](tests/sigma_match/run.sh)）：上面的 Sigma 测试
  证明规则的有效性与编译，而这个测试证明原子规则与关联规则是否真的触发。
  它断言每条原子规则都有代表性的恶意样本事件能触发规则、正常样本事件不会触发
  （例如 `.mitmproxy/` 下的单独 CA 文件不会触发还要求 `_ca/` 目录的记录代理规则）。
  对每条关联规则，它断言一个组在时间窗口内达到阈值的正样本时间线会触发，而
  未达阈值·超出窗口·分组被拆散·缺少分支的时间线保持沉默。解析全部交给 pySigma，
  测试只遍历编译后的条件树与聚合声明，判断哪些事件进入关联规则时使用与原子规则
  相同的匹配器。它把「无法运行的检测规则只是主张」这一原则像 Suricata 那样也应用到 Sigma 一侧。
- **ATT&CK 图层**（[`tests/attack/run.sh`](tests/attack/run.sh)）：确认 ATT&CK 覆盖率图层与
  规则保持一致。打过分的技术与战术必须正好是规则集的 `attack.*`
  标签，且每项技术都必须指向实际存在的规则文件。新增规则而不更新图层
  （或反过来）都会让测试失败。
- **指标依据（source-of-truth）**（[`tests/indicators/run.sh`](tests/indicators/run.sh)）：确认每条规则
  固定的指标是否仍是上游源码发出的那个字符串。查看 `enrich/enrich.go` 的
  `artex-enrich/1.0`、`selfupdate/` 的 `artex-selfupdate`、`guard/guard.go` 的守卫标记、`db/db.go`
  的破坏性令牌是否仍固定在规则中。它抓住其它三个测试漏掉的漂移，也就是所有
  规则都能编译、都能触发，但上游重新同步改掉了 User-Agent 或标记的情况。
  同一个测试还会重新读取机器可读的 [`indicators/artex_indicators.csv`](indicators/artex_indicators.csv)，
  断言已发布的每一行仍依据源码与规则，因此防御者导入的产物也不会
  过期。最后，它断言所读取的每个上游源码都
  出现在 CI 工作流的 `push`·`pull_request` 路径过滤器中，使只改动一个新固定源码的 PR 无法跳过测试、
  让那种漂移通过合并门禁。这样，「不是推测，而是依据本仓库源码中核实过的字符串」（见上）就从一句话
  变成了一道守卫。
- **MISP 导出一致性**（[`tests/misp/run.sh`](tests/misp/run.sh)）：证明 MISP 事件
  （[`indicators/artex_indicators.misp.json`](indicators/artex_indicators.misp.json)）是有效的 MISP
  文档。它由 [pymisp](https://github.com/MISP/PyMISP) 加载，而 pymisp 的对象模型会
  拒绝不存在的属性类型，因此该产物不只是看起来像 MISP，而是确实能被
  导入。测试还断言它与上面的 CSV 逐行同步。相同的值、每个指标意图中的 MISP
  类型·类别，以及如实反映 CSV 的 `to_ids`·`disable_correlation` 标志都必须
  一致（依据规则 = 可处置，因此 `to_ids` 为开；主机取证端口 = 分类提示，因此
  `to_ids` 为关并关闭关联）。该事件与 CSV 并排手工维护。它带有 CSV 没有的说明
  注释·UUID·标签，因此新增、删除或修改 CSV 行时，必须在同一次提交中同时改 MISP 事件，
  二者不一致时这个测试就会失败。
- **Sigma 后端移植性**（[`tests/sigma_backends/run.sh`](tests/sigma_backends/run.sh)）：证明规则
  能超出单个 Splunk 示例完成转换。整棵树（原子 + 关联）能编译到 Splunk、Elasticsearch `eql`
  目标与 Grafana Loki，而五条原子规则在不支持 Sigma 关联的后端（Elasticsearch
  `lucene`、Microsoft `kusto` 后端）上仍能编译。它为下面「验证与转换」中的按后端支持
  表提供了可重新运行的检查。
- **SigmaHQ 约定规范检查**（[`tests/sigma_lint/run.sh`](tests/sigma_lint/run.sh)）：把完整 SigmaHQ 验证器
  （`pySigma-validators-sigmahq` 插件，普通的 `sigma check` 不会加载它们）按
  [`tests/sigma_lint/validators.yml`](tests/sigma_lint/validators.yml) 中记录在案的基线运行，
  并断言问题数为 0。它还确认完整验证器确实运行过、并且只剩有意排除的、记录在案的
  四项检查，因此规则一旦引入新的约定问题（大小写写错的标题、超出分类体系的
  字段），构建就会失败。

除八个规则测试之外，还有两个非规则的测试门禁在同一个 CI 工作流与 [`tests/run-all.sh`](tests/run-all.sh)
中一起运行。一个是测试框架同步检查（确认 run-all.sh·CI·套件目录是否按相同顺序调用相同的套件），
另一个是[主机分类工具](triage/)的 `--self-test`（[`tests/triage-selftest.sh`](tests/triage-selftest.sh)），
它构造合成主机，断言所有分类检查都会触发、并且在干净主机上发现数为 0。

每个脚本只要有一条断言失败就以非 0 代码退出。参见 [`tests/README.md`](tests/README.md)。

## 如何如实地理解这些规则

- **静态指标可以改。** 操作者可以把 User-Agent 设成别的值，因此
  **没有 `artex-enrich/1.0` 或 `artex-selfupdate` 并不代表安全。** 持久的
  信号是*行为*：一个来源从侦察 → 枚举 → 探测 → 认证·注入尝试一路推进，随响应调整、
  不停歇地运行。该层在防御指南（第 1 节·第 2 节·4.1~4.2 节）中说明，上面的
  `sigma/correlation/` 规则把它做成可部署的关联（速度、扇出、守卫拦截突发，以及守卫
  标记+破坏命令的多阶段），而纯 Web 多阶段场景仍然需要按环境定制的基础规则。
- **破坏命令规则属于通用狩猎。** 它对应 ARTEX 守卫的阻止列表，但同样的命令合法
  管理员也会执行。命中只当线索，按自己的环境做允许列表，不要仅凭它
  就断定是 ARTEX。
- **端口指标属于主机取证，不是 Sigma。** ARTEX 服务器默认 `:8787` 与记录代理
  `127.0.0.1:8788`（`cmd/artex/main.go`）最好在可疑主机上用 `ss`·`netstat` 确认。
  因此它没有作为嘈杂的网络规则提供，而是记录在防御指南中，并为分类用途放进
  [指标 CSV](indicators/)。[主机分类脚本](triage/)正是为有 shell 访问但没有 SIEM 的响应人员
  代跑这些主机本地检查（端口、记录代理产物、日志标记、PostgreSQL 模式）的。

## 验证与转换

这些规则已用 [sigma-cli](https://github.com/SigmaHQ/sigma-cli)（pySigma）验证。要复现：

```sh
python3 -m venv .venv && . .venv/bin/activate
pip install sigma-cli

# 结构 + 最佳实践验证（预期：错误 0，问题 0）
sigma check detections/sigma/

# 以这个独立规则集记录在案的基线检查完整 SigmaHQ 约定（预期：问题 0）。
# 上面普通的 `sigma check` 不会加载这些验证器。
pip install pySigma-validators-sigmahq
sigma check --validation-config detections/tests/sigma_lint/validators.yml detections/sigma/

# 编译到目标查询语言，例如 Splunk
sigma plugin install splunk
sigma convert -t splunk --without-pipeline detections/sigma/artex_enrich_user_agent.yml

# 转换整棵树，使关联规则能解析它按 id 引用的原子规则
sigma convert -t splunk --without-pipeline detections/sigma/
```

基线强制所有 SigmaHQ 约定，但 SigmaHQ 单仓库的文件组织体系与分类体系中的四
项检查不适用于这个独立规则集，因此排除。每项排除及其依据都记录在
[`tests/sigma_lint/validators.yml`](tests/sigma_lint/validators.yml) 中，并由上面的规范检查测试
强制执行。

### Sigma 后端移植性

`sigma/correlation/` 规则用 `id` 引用原子基础规则，因此只能在支持
Sigma 关联转换的后端上转换。支持程度因后端而异，所以 `-t` 的选择很重要。下表按固定的
基准（`sigma-cli` 3.1.0，兼容的最新后端）测量，并由
[`tests/sigma_backends/run.sh`](tests/sigma_backends/run.sh) 复现。

- **转换整棵树（原子 + 关联）：** Splunk（`-t splunk`）、Elasticsearch EQL（`-t eql`）、
  Grafana Loki（`-t loki`）。直接转换 `detections/sigma/` 就能同时得到关联查询。
- **只转换原子规则（尚不支持关联）：** Elasticsearch Lucene（`-t lucene`）、
  OpenSearch（`-t opensearch_lucene`），以及面向 Sentinel·Defender XDR 的 Microsoft `kusto`
  后端（`-t kusto`）。在这些后端上转换五条原子规则，关联时间窗口在产品内以原生方式
  表达（例如 Sentinel 计划分析规则的 `summarize ... by bin(TimeGenerated, 30m)`）。传入整个
  目录会因 "Backend does not support correlation rules" 而中断转换。

```sh
# 只转换原子规则，例如 Microsoft Sentinel / Defender（kusto 后端）
sigma plugin install kusto
sigma convert -t kusto --without-pipeline \
  detections/sigma/artex_enrich_user_agent.yml \
  detections/sigma/artex_selfupdate_egress.yml \
  detections/sigma/artex_guard_audit_framing.yml \
  detections/sigma/artex_recording_proxy_ca.yml \
  detections/sigma/destructive_command_hunting.yml
```

在固定版本上已知的边界：Elasticsearch ES|QL 目标（`-t esql`）会拒绝守卫标记规则
（`String value expressions are not supported`），因此在那种情况下请转换其余三条原子规则。另外
IBM QRadar 插件（`ibm-qradar-aql`）与固定的 pySigma 不兼容，需要 `--force-install`，
因此测试不覆盖它。环境中安装的后端可用 `sigma list targets` 查看。

上面的示例使用 `--without-pipeline`，原样输出规则正文中的通用字段名（`cs-user-agent`·`cs-host`·
`CommandLine`）。要匹配自己产品的模式，请去掉该标志并用 `-p` 应用处理
流水线（参见 `sigma list pipelines`）。不过产品流水线在映射字段名的同时，还可能要求规则通用
`logsource` 未指定的目标表。例如 `-p sentinel_asim` 在未为你的数据设置 `query_table` 之前会以
"Unable to determine table name" 中断，因此部署前请把字段与目标表映射到自己的环境。

## 贡献

欢迎贡献检测与加固内容。新规则应让所有指标都依据可观测的事实，在
`description` 中写明局限，干净地通过 SigmaHQ 验证器基线
（`sigma check --validation-config tests/sigma_lint/validators.yml`），并且不含读起来像攻击
指引的内容。参见 [`../CONTRIBUTING.md`](../CONTRIBUTING.md)。
