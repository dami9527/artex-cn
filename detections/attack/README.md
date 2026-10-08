# ARTEX ATT&CK 覆盖率

中文 · [English](README.en.md)

本仓库检测规则所标记的 [MITRE ATT&CK](https://attack.mitre.org/)（Enterprise）技术，整理成
Navigator 图层。它由 [Sigma 规则](../sigma/) 的 `attack.*` 标签手工生成：每项技术都依据一条规则，
而该规则的指标是在本仓库源码中核实过的字符串或行为。没有凭推测加入的条目，[一致性测试](../tests/attack/run.sh) 会保证
图层与规则彼此不出现偏差。

- **`artex_navigator_layer.json`**：ATT&CK Navigator v4.5 格式的图层。

## 分数的含义

这里的覆盖率指「本仓库提供了标记该技术的检测」，而不是「该技术已被完全覆盖」。分数刻意如实反映
检测强度。

- **100：ARTEX 独有签名或行为。** 只存在于 ARTEX 的静态指标（`artex-enrich/1.0`·
  `artex-selfupdate` User-Agent、守卫审计标记），或在它之上建立的行为规则（补全速度·扇出、
  守卫拦截突发）。
- **50–65：通用狩猎线索。** 对应 ARTEX 守卫阻止列表的破坏性命令狩猎。同样的命令合法管理员也会
  执行，因此良性（benign）活动同样会触发。命中只当作线索，不要作为归因的依据。65 表示关联规则
  把该命令与 ARTEX 守卫标记结合、提高了特异性的情形。

## 覆盖的技术

跨六个战术的八项技术。每项技术对应标记它的规则。

- **侦察（Reconnaissance）：T1595 (Active Scanning)、T1592 (Gather Victim Host Information)。**
  [`sigma/artex_enrich_user_agent.yml`](../sigma/artex_enrich_user_agent.yml)、
  [`sigma/correlation/artex_enrich_scan_velocity.yml`](../sigma/correlation/artex_enrich_scan_velocity.yml)、
  [`sigma/correlation/artex_enrich_fanout.yml`](../sigma/correlation/artex_enrich_fanout.yml)，以及
  [Suricata 规则](../suricata/artex.rules)（sid 1000001 / 1000002）。
- **命令与控制（Command and Control）：T1105 (Ingress Tool Transfer)。**
  [`sigma/artex_selfupdate_egress.yml`](../sigma/artex_selfupdate_egress.yml)。
- **执行（Execution）：T1059 (Command and Scripting Interpreter)。**
  [`sigma/artex_guard_audit_framing.yml`](../sigma/artex_guard_audit_framing.yml)、
  [`sigma/correlation/artex_guard_block_burst.yml`](../sigma/correlation/artex_guard_block_burst.yml)、
  [`sigma/correlation/artex_guard_marker_then_destructive.yml`](../sigma/correlation/artex_guard_marker_then_destructive.yml)。
- **影响（Impact）：T1485 (Data Destruction)、T1561.002 (Disk Wipe: Disk Structure Wipe)、T1489 (Service Stop)。**
  [`sigma/destructive_command_hunting.yml`](../sigma/destructive_command_hunting.yml)，其中 T1485 还由
  [`sigma/correlation/artex_guard_marker_then_destructive.yml`](../sigma/correlation/artex_guard_marker_then_destructive.yml) 加强。
- **凭证访问·收集（Credential Access / Collection）：T1557 (Adversary-in-the-Middle)。**
  [`sigma/artex_recording_proxy_ca.yml`](../sigma/artex_recording_proxy_ca.yml)，针对的是 ARTEX 内置流量
  记录器（`traffic/traffic.go`）为解密并记录执行者工具的流量而安装的 MITM 根 CA 产物，属于主机·取证
  狩猎线索。

## 使用方法

1. 打开 [ATT&CK Navigator](https://mitre-attack.github.io/attack-navigator/)。
2. 选择 **Open Existing Layer → Upload from local**，选中 `artex_navigator_layer.json`
   （或指向本仓库的 raw 文件 URL）。
3. 打过分的技术会按检测强度以颜色区分显示，每项技术都带有注释，写明作为依据的规则文件与防御指南
   章节。

## 范围与如实说明

- **覆盖率不等于完整性。** 在这里得分的技术，只表示有规则标记它，不表示该技术的所有变体都能被
  检测。在网络层能被 ARTEX 独有 User-Agent 捕获的信号只有两个：侦察阶段的补全探测器
  （`artex-enrich/1.0`）与攻击阶段的 norma SDK WebFetch（`norma/0.4`），其余攻击流量沿用工具的默认
  指纹。持久的检测来自行为层面
  （参见防御指南 [中文](../../docs/defense-zh.md) · [English](../../docs/defense-en.md) 第 1~2 节·4.1~4.2 节）。纯 Web 多阶段场景仍然
  需要按各自环境定制的基础规则。
- **静态指标可以被改掉。** 操作者可以把 User-Agent 设成别的值，因此没有命中已标记的指标并不代表
  安全。规则文件里也写了同样的注意事项。

## 验证与贡献

请运行[一致性测试](../tests/attack/run.sh)。只需要 Docker，它断言图层打过分的技与战术正好等于规则的
`attack.*` 标签，并且每项技术都依据确实存在的规则文件：

```sh
detections/tests/attack/run.sh
```

新增规则或重新打标签后，请同步更新该图层；规则的技术不在图层中，或图层的技术不在规则中，测试都会
失败。参见 [`../README.md`](../README.md) 与 [`../../CONTRIBUTING.md`](../../CONTRIBUTING.md)。
