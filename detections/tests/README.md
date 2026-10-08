# ARTEX 检测规则测试

中文 · [English](README.en.md)

[`../`](../) 下的检测规则是否真的触发，以及同样重要的、在良性（benign）
流量上是否保持沉默，由本目录可复现的回归测试来证明。无法运行的检测规则
只是一句主张。这些测试把规则文件与防御指南中的主张，变成审查者可以从源码
重新运行的东西。

二进制数据包捕获不入库。抓包**每次运行都确定性地生成**，运行结束后
删除，因此测试以可读源码的形式发布，而不是不透明的固定文件，也不会让仓库膨胀。

## 一次运行所有套件：[`run-all.sh`](run-all.sh)

[`run-all.sh`](run-all.sh) 按与 CI 相同的顺序，用一条命令运行下面八个套件，因此不必手工逐个
调用八个 `run.sh` 脚本。即使前面的套件失败，每个套件也会运行到底；脚本最后输出每个套件一行
PASS/FAIL 摘要，并且只要有一个失败就以非 0
代码退出。

在运行套件之前，先执行测试框架自检（[`check-harness-sync.sh`](check-harness-sync.sh)）。
该自检在下面的套件清单、[CI](../../.github/workflows/detections.yml) 的按套件步骤、磁盘上的套件
目录这三者指向不同套件或不同顺序时，会让运行以失败结束。这是八个套件
自己看不到的唯一空白。只接到三处中一处的套件（例如没有 `run-all.sh` 条目、
只加了 CI 步骤，或哪边都没放进目录）虽然能通过各自的测试，却会让本地为绿的 `run-all.sh`
不再代表绿色的 CI。该自检不是第九个套件，而是一道门禁，因此
不出现在下面的摘要里，检测套件仍然是八个。

```sh
detections/tests/run-all.sh
```

预期输出（节选）：

```
===== detection suites summary =====
  PASS  sigma
  PASS  sigma_match
  PASS  sigma_lint
  PASS  sigma_backends
  PASS  suricata
  PASS  attack
  PASS  indicators
  PASS  misp
RESULT: PASS
```

只要有一个失败就以非 0 代码退出，因此可以直接放进 pre-commit 钩子。现成可用的示例在仓库最上层
的 [`.pre-commit-config.yaml`](../../.pre-commit-config.yaml) 中。
用 `pip install pre-commit && pre-commit install` 安装后，凡是改动检测规则或这些规则所固定的上游源码
文件的提交都会触发运行器，范围与 CI 相同。各套件识别的镜像·版本
覆盖变量（`PYTHON_IMAGE`、`SIGMA_CLI_VERSION`、`SIGMAHQ_VALIDATORS_VERSION`、`SURICATA_IMAGE`）会被运行器
原样继承，因此导出其中任何一个都会同时作用于所有套件。

## Suricata：[`suricata/`](suricata/)

[`suricata/run.sh`](suricata/run.sh) 端到端运行 [`../suricata/artex.rules`](../suricata/artex.rules) 中的
网络规则，并断言五个属性：

- **有效性**：整个规则文件都能用 `suricata -T --init-errors-fatal` 加载，因此即使某条规则不被下面
  任何抓包触发，只要解析·初始化失败也会被抓到。单纯的 `suricata -r` 会跳过这类规则
  并以 0 退出，所以这项加载检查是 Suricata 侧对应 Sigma 套件 `sigma check` 有效性断言的
  东西。
- **存在性（补全探测器）**：sid `1000001` 对每个补全探测恰好触发一次。
- **速度**：超过每来源 300 秒 30 个请求的 `detection_filter` 阈值时，sid `1000002`
  触发。
- **存在性（WebFetch）**：sid `1000003` 对每个 norma WebFetch 请求恰好触发一次，并且在同一抓包中
  补全探测器 sid 保持沉默。确认两个网络签名不仅各自触发，而且彼此
  特异。
- **特异性**：其它条件相同、只把 User-Agent 换成良性（benign）浏览器的抓包，不会产生
  **任何** ARTEX 告警。

[`suricata/gen_pcap.py`](suricata/gen_pcap.py) 用 [scapy](https://scapy.net) 生成抓包：
来自同一固定来源的 N 个独立明文 HTTP 请求/响应流，各自带上指定的 User-Agent，
从固定基准时间戳开始按 1 秒间隔排布。它只写文件，不发送数据包，也不触碰
网络。

### 运行

只需要 Docker。scapy 与 Suricata 都在容器中运行。

```sh
detections/tests/suricata/run.sh
```

预期输出（节选）：

```
  PASS  ruleset loads with zero parse/init errors (suricata -T)
  PASS  sid 1000001 presence: one alert per probe  (got 35, want eq 35)
  PASS  sid 1000002 velocity: fires past 30-in-300s  (got 5, want ge 1)
  PASS  sid 1000003 presence: one alert per WebFetch request  (got 8, want eq 8)
  PASS  enrich sids stay silent on norma traffic (specificity)  (got 0, want eq 0)
  PASS  benign browser UA produces no ARTEX alerts  (got 0, want eq 0)
RESULT: PASS
```

只要有一条断言失败，脚本就以非 0 代码退出，因此可以直接放进 CI 或 pre-commit 钩子。
如果内部有镜像站，请用 `SURICATA_IMAGE` / `PYTHON_IMAGE` 覆盖镜像。

### 为什么速度告警数按不低于下限断言，而不是精确值

`run.sh` 精确断言存在性告警数（`1000001 == 35`·`1000003 == 8`）与良性告警数（`== 0`），
因为它们与引擎版本无关：每个匹配的请求一条告警，其它 User-Agent 则不匹配。速度
规则的告警数取决于具体 Suricata 发行版在边界上如何处理 `detection_filter` 阈值，
因此测试断言 `>= 1`，并单独记录基准值。在 **Suricata 8.0.7** 上，基准
运行对 sid `1000002` 产生 **5** 条告警（越过 300 秒 30 个阈值之后的第 31–35 个流）。

## Sigma：[`sigma/`](sigma/)

[`sigma/run.sh`](sigma/run.sh) 从结构上并用
[sigma-cli](https://github.com/SigmaHQ/sigma-cli)（pySigma）编译验证 [`../sigma/`](../sigma/) 下的 Sigma 规则，
并断言五个属性：

- **有效性**：`sigma check` 对整棵树报告错误 0、条件错误 0、问题 0。
- **编译**：`sigma convert -t splunk` 把整棵树无错误地转换成后端查询语言。
- **指标保留**：每个原子指标字符串（`artex-enrich/1.0`、`artex-selfupdate`、守卫标记，以及记录
  代理 CA 文件名 `mitmproxy-ca-cert.pem`）都原样留在编译后的查询中，因此规则无法
  悄悄丢掉它所依据的字符串。
- **关联规则编译**：[`../sigma/correlation/`](../sigma/correlation/) 的行为规则不会被丢弃，
  而是输出 `event_count` / `value_count` 聚合。
- **关联规则确实起作用**：只*单独*转换一条关联规则会失败，因为它用 `id` 引用原子
  基础规则，而这个引用不是装饰而是强制的。这是 Sigma 侧对应上面 Suricata
  特异性断言的东西。

这把 [`../README.md`](../README.md) 中说明的结构 + 编译验证，变成了可执行、带断言的形态。
下面的配套套件 [`sigma_match/`](sigma_match/) 为原子规则与关联规则都补上*匹配*的那一半：
确认代表性的恶意事件（或时间线）能触发每条规则、正常事件不会触发，因此现在 Sigma 规则也和
Suricata 规则一样，既有可复现的验证测试，也有可复现的匹配测试。（先前担心
手写的粗糙匹配器会削弱规则本身的顾虑，已通过把解析全部交给 pySigma 而消除。信任模型
在下一节说明。）

### 运行

只需要 Docker。sigma-cli 与 splunk 后端在容器中运行，不向仓库写入任何内容。

```sh
detections/tests/sigma/run.sh
```

预期输出（节选）：

```
  PASS  sigma check: 0 errors, 0 condition errors, 0 issues
  PASS  whole tree converts to splunk (exit 0)
  PASS  indicator present: artex-enrich/1.0
  PASS  correlation rule fails to convert alone — it requires its atomic base rule
RESULT: PASS
```

只要有一条断言失败，脚本就以非 0 代码退出，因此可以直接放进 CI 或 pre-commit 钩子。
sigma-cli 固定在基准版本（`3.1.0`）。如果内部有镜像站，请用
`SIGMA_CLI_VERSION` 覆盖版本、`PYTHON_IMAGE` 覆盖镜像。

## Sigma 实时事件匹配：[`sigma_match/`](sigma_match/)

[`sigma_match/run.sh`](sigma_match/run.sh) 证明 [`../sigma/`](../sigma/) 下的 Sigma 规则（包含原子
规则与 [`../sigma/correlation/`](../sigma/correlation/) 的关联规则）确实会在匹配的事件上*触发*，
而在正常事件上保持沉默。它把 Suricata 套件对网络规则给出的「无法运行的检测规则
只是一句主张」这一保证，扩展到了主机·日志层规则。它断言原子规则三项与
关联规则三项，共六个属性：

- **规则与样本配对**：每条原子规则都有 [`events/<名称>.json`](sigma_match/events/) 样本文件，
  每个样本文件都能回溯到规则。没有样本就新增的规则不会在无人验证的情况下溜过去，而是
  会在这里失败。
- **真阳性（true positive）**：每条规则都匹配自己全部恶意样本事件。
- **真阴性（true negative）**：每条规则都不匹配自己任何正常样本事件。例如
  `.mitmproxy/` 下单独的 `mitmproxy-ca-cert.pem` **不会**触发记录代理规则，因为
  该规则的 `|all` 修饰符还要求 ARTEX 使用的 `_ca/` 目录，而证明这一判别
  正是匹配测试的职责。
- **关联规则与时间线配对**：每条关联规则都有 [`events/correlation/<名称>.json`](sigma_match/events/correlation/)
  时间线文件，每条时间线都能回溯到规则。时间线中的每个事件都带表示相对秒数的 `ts`
  字段。
- **关联规则的真阳性**：在一个组于时间窗口内达到阈值的正样本时间线上，每条规则都会触发。
  例如同一来源（`c-ip`）在 10 分钟内扩散到 20 个不同主机的请求会触发补全扇出规则。
- **关联规则的真阴性**：未达阈值、达到阈值但超出时间窗口、分组被拆散、
  时间关联缺少一侧分支的情况都保持沉默。特别是请求量很大但宽度（不同主机数）很小的
  突发**不会**触发扇出规则：信号是宽度而不是量，而证明这一判别
  正是匹配测试的职责。

信任模型如下：解析每条规则的是 pySigma，而不是手写代码。原子规则把修饰符与
条件编译成树（`|contains` → 通配符值，`|all` → AND，`1 of selection_*` → OR），关联规则编译成
聚合声明（类型·group-by·时间窗口·阈值条件·引用的原子规则）。[`check.py`](sigma_match/check.py)
只沿着那棵树与那份声明行走，判断哪些事件进入关联规则时使用与原子规则
完全相同的匹配器，因此权威的 Sigma 逻辑仍然留在 pySigma 内部。遇到未明确
支持的语法时，它不会悄悄放行，而是抛出异常（fail-closed）。范围与局限写在脚本头部。关联规则的
时间窗口采用标准滑动窗口（每个匹配事件都开一个 `timespan` 长度的窗口）的解释，实际 SIEM 的窗口
方式可能不同。匹配**忽略大小写**（这是 `sigma/` 套件所针对的 splunk 后端的
默认行为，破坏命令规则的误报注释本身就以此为前提），关键字匹配是全文子串搜索。
这是对规则字段·值·条件·聚合逻辑的回归测试，并不替代在字段规范化可能不同的各自 SIEM 中
做验证。

### 运行

只需要 Docker。pySigma 在容器中运行，不向仓库写入任何内容。

```sh
detections/tests/sigma_match/run.sh
```

预期输出（节选）：

```
  PASS  rule/sample pairing: 5 atomic rules, 5 event files, no orphans
  PASS  artex_enrich_user_agent: 1/1 positive events matched
  PASS  artex_recording_proxy_ca: 2/2 benign events correctly not matched
  PASS  rule/timeline pairing: 4 correlation rules, 4 timeline files, no orphans
  PASS  artex_enrich_fanout: fired — 20 distinct hosts from one source within the 10-minute window
  PASS  artex_enrich_fanout: quiet — high volume, low breadth: 25 requests from one source but only 4 distinct hosts
RESULT: PASS
```

只要有一条断言失败，脚本就以非 0 代码退出，因此可以直接放进 CI 或 pre-commit 钩子。
pySigma 固定在基准版本（`2.0.0`）。如果内部有镜像站，请用 `PYSIGMA_VERSION`
覆盖版本、`PYTHON_IMAGE` 覆盖镜像。

## Sigma 后端移植性：[`sigma_backends/`](sigma_backends/)

[`sigma_backends/run.sh`](sigma_backends/run.sh) 证明规则能超出 Sigma 测试所运行的单个 Splunk
示例完成转换，并让 [`../README.md`](../README.md) 的按后端支持表保持如实。Sigma 关联规则转换因
后端而异，因此 README 要告诉防御者哪种 `-t` 目标接收整棵树、哪种只接收原子规则。这是
只有重新运行才能相信的主张。它断言两个属性，二者都是正向的，因此只有在真正出现回归时才会失败：

- **关联规则的移植性**：整棵树（原子 + 关联）能在 Splunk、Elasticsearch `eql` 目标与 Grafana
  `loki` 上转换，并且补全指标原样存活到每个查询中。这显示关联规则并非 Splunk 专有。
- **仅原子的回退行为**：五条原子规则也能在 `lucene` 与 Microsoft `kusto` 后端上
  转换。这些后端在固定版本上不支持 Sigma 关联规则转换，因此使用这些后端的
  防御者可以部署原子规则，并用该后端特有的功能表达关联窗口。

它有意不断言「后端 X 处理不了关联规则」这类否定命题，因为那样一来后端
*改进*反而会变成红色构建。如实的局限写在 README 中，本测试的命令
会复现它。[`sigma_backends/check.sh`](sigma_backends/check.sh) 是容器内的那一半：
安装固定的 sigma-cli 与四个后端，读取以只读方式挂载的规则树。

### 运行

只需要 Docker。sigma-cli 与各后端在容器中运行，不向仓库写入任何内容。

```sh
detections/tests/sigma_backends/run.sh
```

预期输出（节选）：

```
  PASS  whole tree (atomic + correlation) converts on 'eql', enrich indicator survives
  PASS  five atomic rules convert on 'kusto', enrich indicator survives
RESULT: PASS
```

只要有一条断言失败，脚本就以非 0 代码退出。sigma-cli 固定（`3.1.0`，
可用 `SIGMA_CLI_VERSION` 覆盖），后端插件安装的是兼容的最新版本。因此这个
套件对上游后端发布最敏感：让支持退化的插件会把构建变红，这就是同时更新
固定版本与 README 表格的信号。

## SigmaHQ 约定规范检查：[`sigma_lint/`](sigma_lint/)

[`sigma_lint/run.sh`](sigma_lint/run.sh) 让 README 与 `CONTRIBUTING.md` 中「干净通过
`sigma check`」的承诺，不仅包含 pySigma 的核心检查，还包含 SigmaHQ 的约定。普通的
`sigma check` 不会加载 `pySigma-validators-sigmahq` 插件，因此标题大小写·字段名分类
体系·日志源分类体系·引用链接约定都不会被检查就过去了。本套件安装该插件，并按
[`sigma_lint/validators.yml`](sigma_lint/validators.yml) 中记录在案的基线运行完整检查集。
它断言两个属性：

- **记录在案的基线是干净的**：与 `validators.yml` 一起运行 `sigma check` 会报告错误 0、问题 0。
- **完整检查集确实在运行，且只剩记录在案的排除项**：不加排除地运行所有 SigmaHQ 验证器时仍会
  报告问题，且每一个都属于 `validators.yml` 有意关闭的四项检查（没有其它）。这是
  防空转的守卫：如果插件加载失败，完整运行什么都不会报告，第一个属性就会
  因为错误的原因通过，因此这里要求已知的排除项必须出现。

四项排除对应 SigmaHQ 单仓库的文件组织方式（带日志源前缀的文件名与 `correlation_`
文件名）与分类体系（通用 `application` 日志源、没有产品名的 `process_creation`），
以及分支链接与永久链接（permalink）的引用约定。它们都不适合一个规模小、
自成一体、且引用本仓库自身活跃文档的规则集。每项排除的依据都写在
`validators.yml` 里。*其余*所有 SigmaHQ 检查都会被强制执行，因此引入新约定问题的规则（大小写
写错的标题、超出分类体系的字段名）会把构建变红。`pySigma-validators-sigmahq` 固定
（`0.21.0`，可用 `SIGMAHQ_VALIDATORS_VERSION` 覆盖），升级版本可能暴露新的约定，这就是
更新规则或记录在案的基线的信号。

### 运行

只需要 Docker。sigma-cli 与验证器插件在容器中运行，不向仓库写入任何内容。

```sh
detections/tests/sigma_lint/run.sh
```

预期输出（节选）：

```
  PASS  sigma check with the documented baseline: 0 errors, 0 issues
  PASS  every reported issue is one of the four documented exclusions
RESULT: PASS
```

## ATT&CK 图层：[`attack/`](attack/)

[`attack/run.sh`](attack/run.sh) 确认 [`../attack/artex_navigator_layer.json`](../attack/artex_navigator_layer.json)
的 [ATT&CK 覆盖率图层](../attack/)与其声称覆盖的规则不出现偏差。与规则
集不一致的覆盖率图层还不如没有，因此这个测试把「这些规则覆盖这些 ATT&CK 技术」
变成审查者可以从源码重新运行的东西。它断言：

- **有效图层**：文件能解析为 JSON，带有 ATT&CK Navigator v4.x 必需字段，并且每个
  条目都有格式正确的技术 ID 与有效的 ATT&CK 战术。
- **双向一致**：打过分的技术与 Sigma 规则的 `attack.*` 技术标签*完全*一致。
  既没有图层缺失的规则技术，也没有规则中不存在的图层技术。战术也以同样方式一致。
- **有依据**：每项打过分的技术的注释都指向实际存在的规则文件，因此图层无法
  引用改名或删除的规则。

这是一致性检查而不是触发测试。它不需要检测后端，只要有 Python 标准库即可，
因此与 Sigma·Suricata 测试不同，没有依赖版本的告警数量。[`attack/check.py`](attack/check.py)
是容器内的那一半：读取以只读方式挂载的检测树，不写入任何内容。

### 运行

只需要 Docker。检查在 Python 容器中运行，不向仓库写入任何内容。

```sh
detections/tests/attack/run.sh
```

预期输出（节选）：

```
  PASS  scored techniques match the rule set exactly (8: T1059, T1105, T1485, T1489, T1557, T1561.002, T1592, T1595)
  PASS  scored tactics match the rule set exactly (collection, command-and-control, credential-access, execution, impact, reconnaissance)
RESULT: PASS
```

只要有一条断言失败，脚本就以非 0 代码退出，因此可以直接放进 CI 或 pre-commit 钩子。
如果内部有镜像站，请用 `PYTHON_IMAGE` 覆盖镜像。

## 指标依据（source-of-truth）：[`indicators/`](indicators/)

[`indicators/run.sh`](indicators/run.sh) 证明上面三个测试做不到的一件事：每条规则
固定的指标是否仍是 ARTEX 自身源码实际发出的字符串。Sigma 测试证明指标经过
规则→查询*编译*后仍然存活，ATT&CK 测试证明图层与规则标签一致，
Suricata 测试证明网络规则在生成的抓包上*触发*。它们都没有回头查看指标所
声称来源的源文件。它们都会漏掉的腐化，就是把探测器 User-Agent
升到 `artex-enrich/2.0`、或改写守卫标记的上游重新同步：规则仍然全部
编译通过，图层仍然一致，pcap 测试也仍然触发，但已部署的规则会悄悄停止
匹配真实的 ARTEX 流量。它针对每个指标做双向断言：

- **源码仍在发出**：值存在于产生它的上游源文件中（`enrich/enrich.go`
  的 `artex-enrich/1.0`、`selfupdate/` 的 `artex-selfupdate`、`guard/guard.go` 的守卫标记）。值
  不存在就意味着上游有规则还没跟上的改动。
- **规则仍在固定**：值存在于以它为基础建立的规则中，因此编辑规则无法把指标从源码
  悄悄剥离。Suricata 规则按 `startswith` 前缀检查，这与该规则
  实际匹配线路的方式相同。
- **阻止列表对应**：破坏性命令令牌（`rm -rf`、`mkfs`、`DROP DATABASE`、`FLUSHALL`）同时出现在 ARTEX
  守卫的阻止列表（`db/db.go`）与映射它的狩猎规则中。它们不是独有指纹，而是通用狩猎
  线索，因此测试只断言规则确实声称的那种对应关系。
- **公开列表保持依据**：逐行重新读取防御者实际取用的产物、即机器可读的指标列表
  [`detections/indicators/artex_indicators.csv`](../indicators/artex_indicators.csv)。所有值都必须仍存在于
  所引用的源文件中、仍固定在所引用的规则里，并且测试确认依据的所有指纹都必须
  出现在该列表中。因此公开的 CSV 无法在任何一个方向上悄悄偏离它声称来源的
  源码。
- **两道关卡都会在固定的每个源码上触发**：测试读取的每个上游源码都包含在运行它的两道关卡中：
  CI 工作流的 `push`·`pull_request` paths 过滤器（[`.github/workflows/detections.yml`](../../.github/workflows/detections.yml)）与
  本地 pre-commit 钩子的 `files` 正则（[`.pre-commit-config.yaml`](../../.pre-commit-config.yaml)）。
  所需集合由指标本身推导，因此固定新源码时（例如以前的 `cmd/artex/main.go` 端口
  就是如此）如果没有在*两道*关卡上都接好，就会在这里失败。否则只改动那个源码的变更会在
  漏掉该源码的关卡上跳过测试：在 CI 上以绿色通过合并门禁，在钩子上则违背
  「与 CI 相同的源码范围」的承诺、在本地始终抓不到。

这把 [`../README.md`](../README.md) 的承诺（「这里的所有指标都不是推测，而是依据本仓库源码中核实过的
字符串」）与 CONTRIBUTING 的第一条贡献契约，变成审查者可以重新运行的守卫。
与 ATT&CK 测试一样，它不需要检测后端，只要有 Python 标准库即可。
[`indicators/check.py`](indicators/check.py) 以只读方式挂载并读取规则树、公开指标列表、固定的源码包，以及
触发它的两道关卡（CI 工作流与 pre-commit 配置），不写入任何内容。

### 运行

只需要 Docker。检查在 Python 容器中运行，不向仓库写入任何内容。

```sh
detections/tests/indicators/run.sh
```

预期输出（节选）：

```
  PASS  enrichment prober User-Agent: 'artex-enrich/1.0' emitted by enrich/enrich.go
  PASS  detections/sigma/artex_enrich_user_agent.yml pins 'artex-enrich/1.0'
  PASS  'FLUSHALL' present in both db/db.go and detections/sigma/destructive_command_hunting.yml
  PASS  enrich-user-agent: 'artex-enrich/1.0' grounded in enrich/enrich.go
  PASS  tested fingerprint 'artex-enrich/1.0' is published in the list
  PASS  .github/workflows/detections.yml push paths covers cmd/artex/main.go
  PASS  .pre-commit-config.yaml files covers cmd/artex/main.go
RESULT: PASS
```

只要有一条断言失败，脚本就以非 0 代码退出，因此可以直接放进 CI 或 pre-commit 钩子。
如果内部有镜像站，请用 `PYTHON_IMAGE` 覆盖镜像。

## MISP 导出一致性：[`misp/`](misp/)

[`misp/run.sh`](misp/run.sh) 处理指标的第二种公开形态，即可直接导入的 MISP 事件
[`detections/indicators/artex_indicators.misp.json`](../indicators/artex_indicators.misp.json)。
上面的指标测试让 CSV 保持以源码为依据，而这个测试让防御者实际载入威胁
情报平台的产物 MISP 事件与那份 CSV 不出现偏差。它断言：

- **确实是 MISP**：事件由 [pymisp](https://github.com/MISP/PyMISP) 加载，而它的对象
  模型会拒绝 `type` 不是真正 MISP 类型的属性。看起来像但无效的类型会
  在这里失败，因此「有效的 MISP」不是一句空话，而是用 MISP 服务器所用的库
  证明的。
- **与 CSV 逐行同步**：每个 CSV 行都对应到恰好一个具有意图中类型·类别的 MISP 属性
  （`http.user-agent` → `user-agent`，守卫标记 `string` → `pattern-in-file`，`port` →
  `port`，`ip-dst|port` → 带合成 `ip|port` 值的 `ip-dst|port`，探查模式 `other` → `other`），并且不会剩下任何没有 CSV 行的 MISP 属性。
  该事件与 CSV 一起手工维护，因此如果在同一次
  提交中没有同步更新 `artex_indicators.misp.json` 就新增·删除·修改 CSV 行的类型，就会失败。
- **`to_ids` 反映 `rule` 列**：有规则支撑的指标是 `to_ids: true`，没有规则的
  主机取证行则是 `to_ids: false` 并带 `disable_correlation: true`。如果把标志翻成与 CSV 含义
  不同就会失败，因此 MISP 事件无法悄悄夸大或缩小哪些指纹可处置。
- **守卫标记按字节保留**，并且 `detections/**` 在 CI paths 过滤器中，因此改动 CSV 或事件
  都会触发本套件。

与上面纯标准库的测试不同，本套件在容器内安装固定的 `pymisp`
（不在宿主机安装任何东西）。[`misp/check.py`](misp/check.py) 以只读方式挂载并读取 CSV、MISP 事件与
CI 工作流，不写入任何内容。

### 运行

只需要 Docker。pymisp 安装在容器中，不向仓库写入任何内容。

```sh
detections/tests/misp/run.sh
```

只要有一条断言失败，脚本就以非 0 代码退出。如果内部有镜像站，请用
`PYTHON_IMAGE` 覆盖镜像、`PYMISP_VERSION` 覆盖固定的库。

## 贡献

新的检测规则在配有一个证明它会触发的测试时更有力。测试应当确定性地
生成自己的输入，对与引擎版本无关的属性做精确断言（更宽松的属性则记为带基准值的下限），并避免
任何可能被读作攻击指引的内容。参见 [`../../CONTRIBUTING.md`](../../CONTRIBUTING.md) 与
[`../README.md`](../README.md) 的规则索引。

八个套件都会在每次触碰 `detections/` 的 push 或 pull request 上通过 CI 运行
（参见 [`../../.github/workflows/detections.yml`](../../.github/workflows/detections.yml)）。而指标
测试还会在它固定的上游源码文件（`enrich/`、`selfupdate/`、`guard/`、`db/`、`cmd/artex/main.go`）
发生变化时运行。因此，让指标退化、与 ATT&CK 图层不一致、在记录在案的后端上
停止转换、破坏 SigmaHQ 约定、与源码失去同步、让 MISP 事件偏离 CSV，或固定一个
工作流尚未监视的新源码的规则改动，都会在合并前把构建变红。
