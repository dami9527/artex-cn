# ARTEX 入侵指标（机器可读）

中文 · [English](README.en.md)

把 ARTEX 自身发出的唯一指纹汇总成一个文件的机器可读列表。它面向需要原子指标本身、而不是检测逻辑的
防御者：把 [`artex_indicators.csv`](artex_indicators.csv) 直接导入威胁情报平台、SIEM 查询表或主机
分类（triage）检查表。所有值都是在本仓库源码中核实过的字符串，也就是[检测规则](../README.md)与防御指南
（[中文](../../docs/defense-zh.md) · [English](../../docs/defense-en.md) 第 2 节）所依据的那些字符串。每一行都写明
该值来自哪里，以及（如果有）建立在它之上的规则。

## 列构成

- **`id`**：指标的稳定 slug。
- **`type`**：指标类型：`http.user-agent`、`string`（在日志·文件中查找的字面量）、`port`、
  `ip-dst|port`、`other`（不属于上述类别的主机产物，例如数据库模式对象名）。
  它们映射到对应的 MISP/STIX 属性类型。
- **`value`**：准确的指标值。包含非 ASCII 守卫标记在内，原样保留。
- **`perspective`**：`target`（从*朝向* ARTEX 所探测系统的流量中观测）或
  `forensic`（在 ARTEX 运行过或经过的主机*之上*观测）。防御指南有意区分这两者，混在一起会得出错误结论。
- **`source`**：发出该值的、以仓库为基准的相对路径源文件（用 `;` 分隔）。这就是
  依据：上游重新同步改变了发出方，这里的指标也必须随之改变。
- **`rule`**：建立在该准确值之上的检测规则（用 `;` 分隔）。不通过（嘈杂的）规则输出、
  而是直接分类的主机取证指标留空。
- **`description`**：一行说明，适用时同时写上如实的注意事项。

## MISP 事件导出

同样的指标也以可直接导入的 [MISP](https://www.misp-project.org/) 事件
[`artex_indicators.misp.json`](artex_indicators.misp.json) 提供。运行 MISP 实例（或使用可导入 MISP
格式的威胁情报平台）的防御者无需手工映射 CSV 列，就能直接导入这些指纹。STIX 2.1 用 MISP 自带的
转换器导出一次即可，因此仓库不再另外手工维护一个有损的第二种格式。

- **类型映射。** 每个 CSV `type` 都成为对应的 MISP 属性类型：`http.user-agent` →
  `user-agent`，守卫标记 `string` → `pattern-in-file`（类别 *Artifacts dropped*），`port` →
  `port`，`ip-dst|port` → `ip-dst|port`（合成值采用 MISP 的 `ip|port` 形式，因此 `127.0.0.1:8788`
  存为 `127.0.0.1|8788`），探索图模式指纹 `other` → `other`（类别 *Other*）。
- **`to_ids` 如实跟随 `rule` 列。** 建立了检测规则的行是可处置的指标，标为 `to_ids: true`。
  没有规则的主机取证行（默认监听端口、回环代理端点，以及探索图模式指纹）是分类提示而不是
  用于阻断的 IoC，因此标为 `to_ids: false` 并带 `disable_correlation: true`（常见端口、`127.0.0.1`
  或通用表名不应污染 MISP 关联）。这与 CSV 的 `rule` 列和下面的注意事项所表达的区分一致。
- **导入。** 用 [pymisp](https://github.com/MISP/PyMISP) 执行
  `MISPEvent().load_file("artex_indicators.misp.json")`，或用 *Add event → Populate from … → MISP
  format* 界面，或 REST API。该事件处于未发布状态并带有 `tlp:clear` 标签。导入时请按自己的
  实例设置分发范围与发布状态。

## 如何如实地理解本列表

- **这些是可以改掉的指纹，不是安全的证据。** 操作者可以把 User-Agent 设成别的值，或更换默认端口，
  因此这里任何一个值*不存在*都**不**意味着没有 ARTEX。持久的信号是行为。参见关联规则与防御指南
  第 1·2·4.1~4.2 节。
- **通用狩猎线索有意排除在外。** 破坏性 shell·数据库命令（`rm -rf`、`DROP DATABASE`……）*不是*
  ARTEX 指纹，合法管理员也会执行。它们是狩猎线索而不是可导入的指标，因此放在
  [`destructive_command_hunting.yml`](../sigma/destructive_command_hunting.yml)与防御指南里，而不在
  本列表中。把它们作为阻断指标导入会产生误报。
- **norma 的 WebFetch User-Agent 是网络签名，不是可导入的原子指标。** 执行者的页面抓取工具在攻击
  阶段发送 `norma/0.4`，Suricata 规则 sid 1000003 会在 `norma/` 前缀上触发，但该字符串是 norma SDK
  中硬编码的自身 User-Agent（`github.com/Autumn-27/norma/tool/webfetch.go`），建立在 norma 之上的
  所有工具都会发出同样的值，并非 ARTEX 独有指纹。把这个值作为阻断指标放进本列表，会让所有 norma
  SDK 流量都报警，这与排除破坏性命令是同一个误报陷阱。因此 norma UA 有意不在本列表中，只通过
  网络规则提供（[`../suricata/README.md`](../suricata/README.md)，sid 1000003）。此外，该值的依据
  不是本仓库源码，而是固定的依赖（`github.com/Autumn-27/norma`），所以下面的依据测试无法像重新读取
  ARTEX 自己发出的字符串那样重新读取它。
- **主机取证端口用于分类而不是阻断。** `:8787` 与 `127.0.0.1:8788` 指向可能正在运行 ARTEX 的
  主机。用 `ss`·`netstat` 确认，不要盲目用防火墙封禁。
- **探索图模式指纹用于数据库排查，不是网络·文件 IoC。** `exploration_nodes`
  表是 ARTEX 存放在 PostgreSQL 中的探索图的核心表。不要凭单次命中下结论，
  而要确认同库中是否有兄弟表（`exploration_edges`·`exploration_anchors`·`assets`·`companies`·
  `activity`）与 `agent_prompts` 种子。操作者可以改表名或删表，因此不存在也不代表安全。
- **`rule` 为空的主机·数据库行有执行器。** 三个不通过 Sigma 规则提供、而是直接分类的指标，也就是
  监听端口、记录代理端点与这个模式指纹，都由[主机分类脚本](../triage/)在可疑主机上全部检查。
  这样只有 shell 访问、没有 SIEM 的响应人员不必手动运行 `ss`·`netstat`·`psql`。

## 验证

本列表由[指标依据（source-of-truth）测试](../tests/indicators/run.sh)覆盖。它重新读取该
CSV，对每一行断言其值仍在所引用的源文件中、仍固定在所引用的规则里，并断言
测试作为依据的所有指标都出现在列表中。凡与源码出现偏差的行，或列表中缺失的已知指纹，都会让测试
失败。运行方式：

```sh
detections/tests/indicators/run.sh
```

MISP 事件由它自己的 [MISP 导出一致性测试](../tests/misp/run.sh)覆盖。它用 pymisp
加载该事件（确保每个属性类型都是服务器接受的实存 MISP 类型），并断言它与本 CSV 逐行
同步，也就是值相同、类型·类别符合意图、`to_ids` 标志与 `rule` 列一致。
该事件与 CSV 并排手工维护。它还带有 CSV 不具备的、按属性整理的注释·稳定
UUID·事件级标签，因此没有会用有损默认值覆盖它们的生成器。新增、删除或修改
CSV 行的类型时，请在同一次提交中同步修改
[`artex_indicators.misp.json`](artex_indicators.misp.json)（新属性要给出新的 `uuid` 与
作为依据的 `comment`）。二者不一致时该测试就会失败，因此不会悄悄忘记更新。运行方式：

```sh
detections/tests/misp/run.sh
```
