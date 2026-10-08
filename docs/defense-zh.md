# 自主 AI 攻击的防御与检测指南

> 本文档面向**防御方**：帮助你理解 ARTEX 这类**自主 AI 渗透智能体**是如何运作的，从而建立**检测并阻断**这类攻击的能力。它不是一份攻击教程。文中所有内容只能用于保护你自有、或已获得明确书面授权的系统。未经授权探测、攻击他人的信息网络本身就是犯罪（参见[顶层 README 的安全与滥用警示](../README.md)）。
>
> English: **[Defense and Detection Guide (defense-en.md)](defense-en.md)**.

自主 AI 攻击工具把过去需要一名操作者手工执行的渗透测试，变成 LLM 智能体**自行拆解目标、真实调用工具、持续累积发现**的 24 小时自动化过程。防御方面对的对手，从「一名熟练攻击者」变成「一群不知疲倦、不会停歇的智能体」。本指南梳理这一变化对检测与响应提出了哪些要求。

---

## 1. 自主 AI 攻击与既有扫描器的区别

传统的漏洞扫描器（例如基于固定签名的工具）把预设的检查项按顺序跑完就结束了。ARTEX 这类自主智能体的结构不同。[系统架构](../README.md#系统架构)一节已经说明：下列要素组合起来，使它能**在无人介入的情况下走完多阶段攻击链**。

- **角色分工的多智能体。** 拆解目标的 `goals`、判定下一步方向并产出意图（intent）的 `planner`、各自把一条意图落到真实工具上的多个 `worker`，以及供人介入的 `mainagent`。planner 是唯一的意图生产者，worker 并行执行这些意图。
- **状态累积在双图上。** 「有什么」（资产图）和「测到哪了」（探索图）分开累积，再用锚点连起来。因此攻击会**逐步加深**：同一资产会被换角度重访，下一步建立在此前的观察之上。
- **事件驱动的闭环。** 图一变，planner 就被唤醒并派发下一批意图；worker 的写入又触发下一轮。这个循环会一直转到目标被证明为止。
- **串行攻击链的稳定推进。** planner 把「找到注入点 → 拿到凭据 → 横向移动 → 提权」这类依赖顺序一次性记进共享 todolist，只在满足前置条件的下一步派发意图。所以即便会话是无状态的，攻击链也不会错序，能够走完。
- **payload 由 LLM 按上下文变形。** 执行工具和 payload 不是写死在代码里的常量，而是 LLM 读上下文后生成的值，因此每一次请求的形态都略有不同。

### 为什么更难检测

- **固定签名不太匹配。** payload 随上下文变化，靠精确匹配已知恶意字符串的规则（WAF 签名）容易被绕过。
- **它可以放慢，也可以像人一样断续推进。** 智能体会在轮次之间停歇，并按发现调整方向，因此只看「短时间内大量请求」的速率型检测会漏掉它。
- **侦察与入侵连成一条流。** 人看到侦察结果、几天后再手工攻击的那段空档消失了，从首次接触到数据外带的时间被大幅压缩。

### 但仍然可以检测：行为难以隐藏

静态指纹（User-Agent、特定 payload 字符串）只要运营者愿意就能改掉。可是**自主智能体的行为形态**是攻击的本质，改起来很难。下面第 2～4 节的重心放在这种基于行为的视角上。

- 单一来源把**多个阶段、性质不同的请求**（侦察 → 枚举 → 认证尝试 → 漏洞利用）**串成一条连续会话**的模式。
- 在人会因疲惫而停下的时段里，仍然**不间断推进**的探测。
- 从失败响应中获取线索、**系统性地变换下一次请求**——不是随机，而是自适应地推进。

---

## 2. 防御者可观测的指纹（IoC 与签名）

指纹分两个视角来看。**(甲) 目标（被攻击方）视角**：从打向你自己系统的 ARTEX 流量里能看到什么；**(乙) 运营者与取证视角**：从 ARTEX 实际运行过（或被入侵后用作跳板）的主机上能看到什么。两者不要混在一起。目标侧能看到的静态指纹有限，真正关键的是行为指纹。

### (甲) 目标视角：打向本系统的流量

- **资产补全（enrich）查询的 User-Agent `artex-enrich/1.0`。** ARTEX 在自动补全资产（DNS、HTTP 校验）时，会直接以这个 User-Agent 向目标发 `GET`（`enrich/enrich.go`）。这条路径与 LLM 无关，由 ARTEX 自己生成，特征是**不跟随重定向、关闭 keep-alive、只读响应开头并取出 `<title>`**，默认并发为 4。因此，如果出现大量以 `artex-enrich/1.0` 为 UA、**短连接、单次 GET、只读标题就断开**的查询同时打向多个资产，就强烈指向 ARTEX 系的补全流量。不过运营者可以改掉 UA，所以**它不出现并不代表安全。**
- **真正的攻击流量跟随工具自身的默认指纹。** worker 通过真实工具（用 Bash 调起的外部工具、HTTP 请求）发出请求。ARTEX 只会在子进程环境变量里注入 `HTTP_PROXY` 和代理 CA 路径，好把流量引入自带的记录代理，**并不会给攻击流量强制加上 ARTEX 专属的 User-Agent**。所以目标看到的 User-Agent 与请求头是**当时所用工具的默认值**（各类命令行工具的默认 UA）。运营者没定制就留下常见自动化工具的指纹，定制过则可能伪装成正常浏览器。因此**不要依赖单一 UA 匹配，要和基于行为的检测结合**。
- **worker 内置的 WebFetch 工具会留下 `norma/0.4` User-Agent。** 与上面经 Bash 调起的外部工具不同，ARTEX 通过 norma SDK（`github.com/Autumn-27/norma`）直接发起的 HTTP 查询（WebFetch 工具）会在攻击阶段带上该 SDK 的默认 User-Agent `norma/0.4`。在明文 HTTP 上（或 TLS 终止点）可在网络层观测到，[Suricata 规则 sid 1000003](../detections/suricata/README.md)会匹配这个前缀（`norma/`）。但这个 UA 并非 ARTEX 独有，使用 norma SDK 的其他工具也一样，因此和 `artex-enrich/1.0` 一样只是辅助线索而非独立证据；而经 Bash 调起的工具各用自己的 UA，所以**它不出现并不代表安全。**
- **没有内置限速。** ARTEX 自身没有针对目标流量的速率限制，请求速度由 LLM 调起的外部工具决定。不过每个任务的 worker 默认并行 3 个，同一目标上**可能同时推进多条意图**。也就是说，它既可能表现为「缓慢的单会话」，也可能表现为「多角度并发」，用单一固定阈值很难抓。
- **行为签名（最重要）。** 下列模式**同时出现**才指向自主智能体。
  - 单一来源（或少数轮换来源）在**很短的间隔内连环出现侦察 → 目录与端点枚举 → 参数探测 → 认证与注入尝试**。
  - 打向同一端点、但**会依据响应码与长度系统性地变形**的连续请求（是自适应，不是随机 fuzz）。
  - 越过人类作息时段、**长时间不间断**的会话。
  - 在失败（401/403/429）之后仍不停手、继续尝试**绕过变形**的执拗。

### (乙) 运营者与取证视角：ARTEX 运行过的主机

在入侵调查中，可用于在跳板、中转主机上寻找 ARTEX 被安装、被运行的痕迹。

- **默认监听端口 `:8787`。** ARTEX 服务端的默认 HTTP 地址（`cmd/artex/main.go`，可用 `--addr` 改）。内网主机在这个端口上开着管理界面（仪表盘、任务、资产图），就是怀疑存在 ARTEX 实例的依据。
- **记录型 MITM 代理 `127.0.0.1:8788`。** 用于拦截 worker 的 Bash 与 HTTP 执行、并记录全过程的本地代理默认地址（`cmd/artex/main.go` 的 `--proxy` 默认值，仅监听回环）。它会自建 CA 解密并记录 TLS（`mitmproxy-ca-cert.pem`），因此线索是：主机上是否存在**由 ARTEX 安装的受信任 CA 证书**，以及是否存在向子进程注入 `HTTP_PROXY` 与代理 CA 路径环境变量的痕迹。这种注入会进入 ARTEX 生成的每一个 worker 工具，且变量名硬编码在源码里（`agent/worker.go`），所以**运行中的进程是否同时带有代理变量与 mitmproxy CA 信任变量**，比单个端口更具特异性。[主机分类工具](../detections/triage/README.md)会在 `/proc`（取证镜像里则是抓到的环境变量转储）中核对这一组合。
- **自更新回调 `artex-selfupdate`。** 自动更新查询 GitHub Release 时使用的 User-Agent（`selfupdate/`）。在出站（egress）日志里看到以该 UA 打向代码托管站点的请求，就说明主机上存在 ARTEX 二进制。
- **PostgreSQL 上的双图。** 库里存在 `exploration_nodes`、`assets`、`companies`、`activity` 这类表，以及 `agent_prompts` 种子数据，是 ARTEX 数据存储的特征。
- **基于数据库的正则审批规则与审计日志。** 评估工具调用的 intercept 规则存放在数据库中，按优先级用正则评估（`intercept/`）。被拦截的调用会连同以 `【ARTEX 平台管控·非目标防御】` 开头的管控框架文本一起写进审计日志（`GET /api/audit`），所以在被入侵主机的审计记录里看到这个字符串，可以佐证 ARTEX 的运行。
- **破坏性命令的狩猎指标。** ARTEX 自带守卫所内置、作为拦截目标的命令模式，正好是自主智能体**可能尝试**的命令集合的逆像。在主机命令审计中，可把下列内容作为狩猎指标：`rm -rf`、`mkfs`、`dd of=/dev/`、`shred`/`wipe`、SQL 的 `DROP DATABASE`/`DROP TABLE`/`TRUNCATE`、MongoDB 的 `drop`/`dropDatabase`、Redis 的 `FLUSHALL`/`FLUSHDB`、`curl`/`wget` 的 `-X DELETE`，以及 `curl … | nc …` 这类数据外带管道。不过最后的外带管道并不是破坏数据的命令，而是把数据带出去的泄漏信号，性质不同。与前面的破坏模式不同，ARTEX 守卫对这条规则也只在默认关闭的状态下内置（因为正常的排查用反弹 shell 或数据传输管道误报较多），而可部署的破坏性命令狩猎规则（[`destructive_command_hunting.yml`](../detections/sigma/destructive_command_hunting.yml)）也只限定在破坏范围内、不含该模式；所以外带管道不要直接拦截，作为单独的狩猎指标按自己的环境调优。

> 小结：**目标侧防御押在行为指纹上，静态 UA（`artex-enrich/1.0`、`norma/0.4` 等）只作辅助线索**。运营者与取证指纹（`:8787`、`127.0.0.1:8788`、`artex-selfupdate`、数据库 schema、审计日志框架文本）在**排查被入侵的中转主机**时有效。

### 为什么封 IP 是一种很弱的初级防御

安全事件一曝光，社交平台与社区里就常出现「共享攻击 IP 列表，去防火墙上封掉」的帖子。且不说分享者的善意，**把来源不明的非官方 IP 列表直接写进拦截规则并不值得推荐**，在自主 AI 攻击面前尤其如此。

- **来源难以验证。** 个人发布的非官方列表无从确认是谁、依据什么收集的，混进了与事件无关的 IP 也没有办法筛掉。
- **很快就过期。** 自主智能体会经 VPN、云主机、被劫持的中转服务器频繁更换出口 IP（见上面 (甲) 一节的「少数轮换来源」）。昨天观测到的攻击 IP 今天很可能已被弃用，封了列表攻击者照样换 IP 回来。
- **误封风险很大。** 列表里混入共享网段、CDN、正常云厂商 IP 时，一封就会连带切断正常客户流量或内部服务。与自动拦截（第 6 节）结合时，误封的损害扩散得更快。

这并不是说封 IP 毫无用处。**从响应机构或可信的威胁情报获取官方威胁指标（IoC），评估有效期与误封可能之后再落地**，它才有意义。但封一行 IP 只是追赶轮换出口的临时措施；真正持久的是难以改变的**行为**（第 2～4 节的行为检测）与**攻击面收敛**（第 3、5 节的加固：清理暴露资产、打补丁、上 MFA）。「指纹可以改，行为难以藏」这个前提在此处同样成立。

---

## 3. 攻击者觊觎的入口与加固

自主智能体与人攻击者盯着**同样的弱点**，只是更快、更执拗地反复尝试。以下是防御视角下应优先加固的点。

### 3.1 对外暴露的攻击面与已知（n-day）漏洞

自主智能体最先、也最稳定盯上的入口，不是精巧的 0-day，而是**暴露在外部、尚未修补的已知漏洞**。典型目标包括边界设备（VPN、防火墙）、对公网开放的管理与运维控制台、应用服务器与中间件、框架（例如被广泛利用的 WebLogic、Struts 系），以及**并非主站、而是为合作方、招聘或员工接入的辅助系统**。自主智能体会用资产图自动枚举暴露面，然后在补丁落地之前，用公开的 PoC 同时对数百个资产打 n-day。它在这件事上的速度与人攻击者之间形成了明显的不对称。

- **收敛攻击面。** 持续维护对公网暴露的资产、管理控制台、辅助系统的清单，不必放在外网的一律挪到内网、VPN 或允许列表之后。
- **尽快修补已知漏洞。** 边界设备、Web 服务器、应用服务器、中间件的公开漏洞（n-day）是自主智能体的首选目标，因此对已有公开 PoC 的组件要把补丁窗口压到最短。先修哪些，可以把收录了已观测实际利用漏洞的 [CISA KEV（已知被利用漏洞）目录](https://www.cisa.gov/known-exploited-vulnerabilities-catalog)作为优先级输入，并与 7.1 的官方通告交叉确认。与其在文档里写死某个 CVE 编号，不如以这类持续更新的官方清单为准，这样自主智能体转向新的 n-day 时也不会落后。
- **收紧不得不暴露的接口。** 对必须对外开放的管理与运维接口，叠加访问来源限制（IP 允许列表）、MFA、VPN，阻断未认证的枚举本身。
- **用与主站相同的标准管理辅助系统。** 合作方、招聘、员工用的辅助系统也按与主站同等的补丁与监控水平管理。自主智能体钻进去的入口，往往是这些辅助路径而不是主站。认证流程本身的加固见下面的 3.2。

从检测角度看，如果针对某个漏洞已知路径（URL、参数）的请求从外部以很短间隔连环出现，就是 n-day 扫描的信号。这个信号与第 2 节的行为指纹、第 4 节的同源多阶段关联规则结合时最明显。

### 3.2 辅助认证与身份核验流程

附加服务、合作方、招聘渠道这类**走主站之外路径接入的认证与身份核验流程**，校验往往更松，因而成为绕过的目标。自主智能体会自动枚举这些路径，读取响应差异，系统性地寻找绕过条件。

- 把身份核验与认证环节的强度**统一到与主站同级**，并逐一排查可绕过的辅助路径。
- 认证状态迁移（未登录 → 已登录，普通 → 高权限）**在服务端重新校验**，不要直接相信客户端送来的信任标记（Cookie、请求头、参数）。
- 检查身份核验令牌与一次性验证码的**有效期、可重用性与可猜测性**。

### 3.3 API 认证与授权（IDOR、越权）

- 对所有对象访问强制**服务端归属与权限校验**（阻断只改标识符就能打开他人资源的 IDOR）。
- 逐一排查横向与纵向越权路径。自主智能体会机械化地增减标识符并大批量尝试，因此**单件手工测试容易漏掉的窟窿**很快就会被找出来。

### 3.4 撞库（credential stuffing）

把泄露的账号密码列表拿去逐一尝试，会被自主智能体以**速度与分布式**放大。

- 在登录与身份核验端点上加**自适应限速**（基于 IP、账号、设备、行为）。
- 对敏感操作强制**多因素认证（MFA）**。撞库即使猜中密码，第二因素仍然挡得住。
- 用**凭据泄露检测**（比对已知泄露列表、识别异常登录位置与速度）提前拦截。
- 对登录失败与成功分布的**突变**（突然的低速大范围尝试）告警。

### 3.5 会话、令牌与密钥管理

- 把会话令牌的**作用域、有效期、续期**压到最小，敏感状态迁移时要求重新认证。
- 不要把 API 密钥、内部令牌**暴露在响应、日志或错误信息里**（自主智能体会积极从错误响应里收集线索）。

---

## 4. 检测规则与日志模式（实战）

这里用不绑定具体产品的**伪规则**形式给出，请自行翻译成你所用的 WAF、IPS、SIEM 语法。其中基于静态指纹的规则，以可直接部署的 [Sigma 规则（`detections/sigma/`）](../detections/README.md)形式提供。核心的行为与关联检测（4.1、4.2）虽然无法还原成单条规则，但其中已用 ARTEX 源码核实过依据的行为指标，仍以可部署的 [Sigma 关联规则（`detections/sigma/correlation/`）](../detections/README.md)形式提供（补全查询速率、补全查询目标数量、守卫拦截突发、守卫标记与破坏性命令在同一主机上同时出现）。至于纯 Web 的多阶段关联（枚举 → 探测 → 认证），其流量无法归结为单一的 ARTEX 专属 UA，因此需要按环境编写基线规则。该关联已作为下面的 4.2 中可直接部署的通用行为基线模板给出，请按自己的 SIEM 与基线调整后作为起点使用。网络层在明文 HTTP 上（或 TLS 终止点）能观测到的 ARTEX User-Agent 有两个，都提供为 [Suricata 规则（`detections/suricata/`）](../detections/suricata/README.md)：一个是补全探测的 `artex-enrich/1.0`（存在签名 sid 1000001、高速枚举变体 sid 1000002），另一个是 norma SDK 的 WebFetch 工具在攻击阶段发出的 `norma/0.4`（sid 1000003）。

### 4.1 WAF 与 IPS（基于行为）

- 单一来源在**一条会话里**接连发出**性质不同的请求群**（静态资源占比低，枚举、参数探测、认证尝试占比高）时加分。
- 对**随响应码与正文长度变形**的连续请求（熵高但不是随机，而是自适应模式）加权。
- 把 `artex-enrich/1.0` 这类已知自动化 UA **立即标为高风险**，但不要把 UA 缺失解释为安全。

### 4.2 SIEM 关联规则

- **同源多阶段关联**：同一 IP、ASN 或会话在**很短的时间窗内同时**出现 (a) 目录与端点枚举、(b) 参数探测、(c) 认证或注入尝试，则告警「疑似自主攻击」。
- **时段异常**：偏离服务正常流量分布、**长时间不间断**的单一会话。
- **失败后继续**：收到 403、429 仍不停手、继续做**绕过变形**的来源。

下面给出**同源多阶段关联**的可直接部署基线模板。由于攻击流量没有 ARTEX 专属 User-Agent，这份模板与 `detections/sigma/` 中基于 ARTEX 源码的规则不同，它是**通用行为规则**：不看某个攻击工具的指纹，只看「一个来源在短窗内同时完成枚举、探测与认证」这一行为。文件里包含三条子规则，以及只有同一客户端在时间窗内同时满足三者才触发的 temporal 关联规则。

```yaml
# ── 通用行为基线模板（不是 ARTEX 专属签名）──
# 对应防御指南 4.2 节的同源多阶段 Web 模式（枚举 → 探测 → 认证）。
# ARTEX 攻击流量不带 ARTEX 指纹，因此与 detections/sigma/ 下的规则不同，
# 这是一份没有用 ARTEX 源码固定依据的通用行为起点。字段名（SigmaHQ 的
# Web 服务器分类）与阈值、时间窗，务必按自己的日志与基线调整。
# 自足型：三条子规则 + 只有同一客户端在窗口内同时满足三者才触发的
# temporal 关联规则。
title: Web Endpoint Enumeration Burst From One Source
id: f03c360c-dc33-4a8a-afa8-821b1ff5c4e3
status: experimental
description: |
    Stage 1 of the same-source multi-stage pattern in the ARTEX defense guide section 4.2: a
    burst of endpoint or directory enumeration from a single client, seen as a high rate of 404
    and 400 responses in a short window. This is generic behaviour, not an ARTEX-specific
    signature; tune the count and window to your own baseline. On its own this leg is low signal
    and earns weight only inside the correlation below.
references:
    - https://github.com/dami9527/artex-cn/blob/main/docs/defense-zh.md
    - https://github.com/dami9527/artex-cn/blob/main/docs/defense-en.md
author: artex-cn defense guide (generic template)
date: 2026-10-07
tags:
    - attack.reconnaissance
    - attack.t1595
logsource:
    category: webserver
detection:
    enum_misses:
        sc-status:
            - 404
            - 400
    condition: enum_misses
falsepositives:
    - Broken links, authorised vulnerability scanners, or misconfigured clients that generate
      many 404 responses.
level: low
---
title: Web Parameter Or Path Injection Probe From One Source
id: f9296e55-6b7a-4030-b5f7-5f7b146233be
status: experimental
description: |
    Stage 2 of the same-source multi-stage pattern: parameter or path probing, matched here as
    query strings carrying common injection or traversal markers. This leg is unavoidably
    signature-like and noisy on its own, so it is scored low and earns weight only inside the
    correlation below. Extend the marker list to your own probe corpus and WAF categories; it is
    a coarse proxy for the broader "adapts requests to responses" behaviour the guide describes.
references:
    - https://github.com/dami9527/artex-cn/blob/main/docs/defense-zh.md
    - https://github.com/dami9527/artex-cn/blob/main/docs/defense-en.md
author: artex-cn defense guide (generic template)
date: 2026-10-07
tags:
    - attack.initial-access
    - attack.t1190
logsource:
    category: webserver
detection:
    probe_markers:
        cs-uri-query|contains:
            - '../'
            - "' or "
            - ' union select '
            - '<script'
            - '; drop '
    condition: probe_markers
falsepositives:
    - Legitimate request payloads that resemble probe markers; tune the marker list to your
      application.
level: low
---
title: Authentication Or Identity-Verification Attempt From One Source
id: 2614cacb-7455-46ad-9af2-7b9633f12d7b
status: experimental
description: |
    Stage 3 of the same-source multi-stage pattern: requests to login, authentication, or
    identity-verification endpoints, or 401 and 403 responses. Map the paths and your own
    authentication-event fields to your application; auxiliary, affiliate, and broker channels
    often expose weaker identity-verification endpoints than the main service and belong here
    too. This leg is broad by design and is only meaningful inside the correlation below.
references:
    - https://github.com/dami9527/artex-cn/blob/main/docs/defense-zh.md
    - https://github.com/dami9527/artex-cn/blob/main/docs/defense-en.md
author: artex-cn defense guide (generic template)
date: 2026-10-07
tags:
    - attack.credential-access
    - attack.t1110
logsource:
    category: webserver
detection:
    auth_path:
        cs-uri-stem|contains:
            - '/login'
            - '/auth'
            - '/verify'
            - '/otp'
    auth_deny:
        sc-status:
            - 401
            - 403
    condition: auth_path or auth_deny
falsepositives:
    - Ordinary users signing in; this leg is broad and only meaningful inside the correlation.
level: low
---
title: Same-Source Multi-Stage Web Attack (Enumeration, Probe, Auth)
id: 9b7c7b86-702f-42b8-be99-3e60a188ec5b
status: experimental
description: |
    The behaviour-based core of ARTEX defense guide section 4.2 as a deployable template: one
    client runs endpoint enumeration, parameter or path probing, and an authentication or
    identity-verification attempt within the same short window. This is the pattern an autonomous
    agent drives at machine speed and keeps driving past 403 and 429 responses. It is UA-free and
    carries no ARTEX fingerprint, so it is a GENERIC behavioural rule, not one of the
    ARTEX-source-grounded rules under detections/sigma/. Normalise the client field (c-ip, or a
    session identifier if you have one) and tune the window to your baseline. If three legs are
    too strict and miss cases, relax to any two of the three.
references:
    - https://github.com/dami9527/artex-cn/blob/main/docs/defense-zh.md
    - https://github.com/dami9527/artex-cn/blob/main/docs/defense-en.md
author: artex-cn defense guide (generic template)
date: 2026-10-07
tags:
    - attack.initial-access
    - attack.t1190
correlation:
    type: temporal
    rules:
        - f03c360c-dc33-4a8a-afa8-821b1ff5c4e3
        - f9296e55-6b7a-4030-b5f7-5f7b146233be
        - 2614cacb-7455-46ad-9af2-7b9633f12d7b
    group-by:
        - c-ip
    timespan: 10m
falsepositives:
    - An authorised vulnerability scan or QA run from a single source; allow-list its address.
level: high
```

使用这份模板时请注意几点。

- 这个代码块已用与检测包相同的工具验证过：`sigma check` 在 SigmaHQ 全量规范下通过（错误与问题均为 0），`sigma convert -t splunk` 也能生成查询（三条子规则在 10 分钟窗口内按 `c-ip` 分组，三者齐备才触发）。但它无法用 ARTEX 源码固定依据，所以没有放进 `detections/` 那棵受测试覆盖的规则树里——为的是守住那棵树「只收录源码核实过的东西，不做推测」的原则。
- 这份模板是关联（correlation）规则，`sigma convert` 究竟是导出整份模板还是只导出三条子规则，取决于后端是否支持 Sigma 的关联转换。用同一固定版本（`sigma-cli` 3.1.0）实测：整份模板能在 Splunk（`-t splunk`）、Elasticsearch EQL（`-t eql`）、Grafana Loki（`-t loki`）上转换；而 Microsoft `kusto` 后端（Sentinel、Defender）与 Elasticsearch Lucene（`-t lucene`）不支持关联规则转换（`Backend does not support correlation rules`），此时只能转换三条子规则，「10 分钟窗口、同一 `c-ip`」的关联要在产品里自行表达（例如 Sentinel 计划分析规则里的 `summarize ... by bin(TimeGenerated, 10m), <客户端>`）。这与检测包记录的移植性一致，实测支持表见 [Sigma 后端移植性](../detections/README.md#sigma-后端移植性)。
- 第 2 阶段（探测）依赖注入与遍历标记列表，是较粗的信号，单独使用误报很多。所以三条子规则的 `level` 都设得很低，只有三者在同一来源同时出现时，关联规则才升级为高等级告警。
- 客户端按 `c-ip` 分组。如果前面有代理或 CDN，请换成用 `X-Forwarded-For` 还原出的真实客户端地址，或会话标识符。如果要求三个阶段全部满足过于严格、出现漏报，可放宽为三者中满足两者即触发。

### 4.3 认证日志

- 账号或 IP 维度的**登录失败率突变**、跨大量账号的**低速分布式尝试**（撞库的典型特征）、失败转成功的**异常迁移速度**。
- 针对身份核验与一次性验证码端点的**枚举式访问**。

### 4.4 出站与取证

- 内网主机以 `artex-selfupdate` UA 打向代码托管站点的请求。
- 内网中绑定 `:8787`（管理界面）与 `127.0.0.1:8788`（记录代理）的进程。
- 以 `artex-enrich/1.0` UA 在短时间内大量查询外部资产的 DNS 与 HTTP 补全模式。

---

## 5. 加固检查清单

供防御团队直接对照检查。

- [ ] 登录、身份核验与敏感 API 已应用**自适应限速**（IP、账号、设备、行为）。
- [ ] 敏感操作强制 **MFA**。
- [ ] 所有对象访问都有**服务端归属与权限校验**（阻断 IDOR）。
- [ ] 认证状态迁移**在服务端重新校验**，不直接相信客户端送来的信任标记。
- [ ] **辅助、合作方、招聘渠道**的身份核验强度已与主站统一。
- [ ] 已运营泄露凭据的**检测与比对**。
- [ ] WAF 以**行为模式**运行，不只依赖固定签名。
- [ ] **不直接套用来源不明的 IP 封禁列表**，而是从可信来源获取官方威胁指标（IoC），评估有效期与误封风险后再落地。
- [ ] SIEM 中已加入**同源多阶段关联规则**。
- [ ] 认证、访问、出站**日志保留了足够长的周期**（自主攻击速度很快，事后追溯的资料至关重要）。
- [ ] 持续维护对公网暴露的资产、管理控制台与辅助系统清单以**收敛攻击面**，并及时**修补边界设备、应用服务器与中间件的已知（n-day）漏洞**。
- [ ] 通过网络**分段**缩小横向移动与提权的爆炸半径。
- [ ] 密钥与令牌**不暴露在响应、日志、错误信息中**。
- [ ] 已准备**自动拦截与隔离**的响应能力（只等人来批准，跟不上自主攻击的速度）。

---

## 6. 事件响应要点

自主 AI 攻击的特点就是**速度**：一名攻击者用一年完成的渗透与外带，智能体可以在短得多的时间里走完。响应设计也要以这个速度为前提。

- **把自动拦截前置。** 隔离可疑来源、使会话失效、骤降速率这类动作，要能在人工批准之前**自动触发**。把所有动作都串进人工审批回路，就跟不上攻击速度。
- **提前确定要保留哪些日志。** 认证日志、访问日志（在允许范围内含请求正文）、出站日志、DNS 查询。自主攻击会很快堆积痕迹，要事后还原攻击链就必须靠这些资料。
- **按资产粒度追踪影响范围。** 攻击会沿资产图蔓延，因此必须还原从最初入口资产到横向移动、提权的**整条路径**，才能防止二次入侵。

### 6.1 可疑主机与流量的分类（triage）流程

怀疑与 ARTEX 有关时，按顺序给出最先要确认的内容。与第 2 节把指纹分成两个视角一样，分类也分成**(甲) 我的服务是否成为目标**和**(乙) 某台主机上是否运行过 ARTEX**。无论哪一步，都不要凭单项命中下结论，要看多项指标与行为信号是否同时出现。即使静态指标全都没有，只要看到行为信号就继续查下去。

**(甲) 目标侧：我的服务是否成为 ARTEX 的目标**

1. 在访问日志与认证日志中查询补全查询 User-Agent `artex-enrich/1.0`。确认是否存在不跟随重定向的单次 `GET`、以很短间隔同时打向多个资产的查询（第 2 节 (甲)）。运营者可以改掉 User-Agent，所以即使没查到也继续下一步。
2. 寻找来自同一来源（或少数轮换来源）的**多阶段连环**：从侦察到端点枚举、参数探测、认证与注入尝试在很短间隔内接连发生，会随响应码与长度自适应，且在 401、403、429 之后仍不停止绕过变形。这种**行为信号比静态 User-Agent 留存更久**（第 2 节 (甲) 行为签名）。
3. 如果你们有 SIEM，把这一行为接成 [Sigma 关联规则](../detections/README.md)（补全查询速率、目标数量、守卫拦截突发、守卫标记与破坏性命令同时出现），并按上面「把自动拦截前置」的原则，把命中的来源列入隔离与使会话失效的对象。

**(乙) 主机取证：某台主机上是否运行过 ARTEX**

在可疑主机上确认下列内容。指标依据在第 2 节 (乙) 与机读的[威胁指标列表](../detections/indicators/artex_indicators.csv)。下面五项只读检查中，前四项（监听端口、出站日志、审计日志、状态与记录存储）可由[主机分类脚本](../detections/triage/)（`detections/triage/artex_host_triage.py`）一次性代跑。第五项命令审计不放进自动指标——破坏性命令并非 ARTEX 独有指纹，正常管理员也会用，因此它是狩猎线索，脚本不代跑，需要在主机命令历史里人工比对。没有 SIEM、只有 shell 访问时可以先跑一遍，每一项命中都按下文说明只当作线索。

1. **监听端口。** 在主机上直接确认默认服务端口 `:8787` 与回环记录代理 `127.0.0.1:8788` 是否处于监听状态。
   ```sh
   ss -ltnp | grep -E ':8787|:8788'   # 没有 ss 时用 netstat -ltnp
   ```
   这两个端口可以用 `--addr`、`--proxy` 改掉，所以即使这条查询为空，也要一并看看全部监听端口以及是否有内部管理界面在运行。
2. **出站日志。** 确认出站日志里是否有以自动更新 User-Agent `artex-selfupdate` 打向代码托管站点（GitHub Release）的请求（`selfupdate/`）。这可以说明该主机上运行过 ARTEX 二进制。
3. **审计日志。** 审计记录里如果有守卫管控标记 `【ARTEX 平台管控·非目标防御】`，可佐证 ARTEX 的运行（`guard/guard.go`）。每一次被拦截的工具调用都会以这段框架文本留下记录。
4. **状态与记录存储。** ARTEX 把探索图存放在 PostgreSQL（`exploration_nodes`、`assets`、`companies`、`activity` 表以及 `agent_prompts` 种子），并在可执行文件旁的数据目录（`cmd/artex/main.go` 的 `--data` 默认值）里留下状态与记录。该目录下直接有按任务存放产物的 `tasks/` 与存放会话记录的 `transcripts/`，而记录代理的产物单独收在其中的 `traffic/` 子目录里（`server/manager.go` 会把数据目录下的 `traffic/` 作为记录代理的存储打开）。因此受信任 CA 证书位于 `traffic/_ca/mitmproxy-ca-cert.pem`，流量索引位于 `traffic/_index/index.sqlite`，记录下来的请求与响应正文位于 `traffic/_blobs/`。这三者与 `tasks/`、`transcripts/` 同时出现，就更能说明记录代理确实运行过。
5. **命令审计。** 把破坏性命令狩猎指标（第 2 节 (乙) 末尾的 `rm -rf`、`DROP DATABASE`、`FLUSHALL`、外带管道等）与主机命令历史比对。正常管理员也会用同样的命令，因此只作线索。

静态指标（端口、User-Agent、标记）可能被运营者改掉或清除。所以**它不出现并不代表安全**；把 (甲) 的行为信号与 (乙) 的主机痕迹合起来判断，才是自主 AI 攻击分类的关键。

---

## 7. 官方渠道：威胁指标、安全通告与报告义务

防御团队应当从官方渠道获取威胁指标与安全通告，出事后还要履行法律规定的报告义务。请以下列官方来源为第一依据，而不是流传的非官方列表。

### 7.1 获取威胁指标与安全通告的渠道

- **国家互联网应急中心（CNCERT/CC）**：<https://www.cert.org.cn>，发布安全通告、漏洞公告与事件响应信息，并通过网络安全威胁信息共享机制在成员单位之间共享威胁情报。
- **国家互联网信息办公室**：<https://www.cac.gov.cn>，发布网络安全与数据安全相关的政策、通报与个人信息保护要求。
- **12377 举报中心**：<https://www.12377.cn>，受理网络违法犯罪与不良信息举报。
- **金融行业**：由国家金融监督管理总局及其指导下的行业信息安全共享机制发布行业通告。

第 5 节加固清单里「从可信来源获取官方威胁指标」指的就是这些渠道。即便是官方指标，也要先评估有效期与误封风险再落地，原则与第 2 节「为什么封 IP 是一种很弱的初级防御」所述相同。

### 7.2 法律上的报告义务

自主攻击蔓延很快，因此请把法定报告环节预先写进第 6 节的事件响应流程。以下是要点，具体适用范围、时限与要件须以主管部门最新规定为准。

- **个人信息泄露**：依据《个人信息保护法》，发生或者可能发生个人信息泄露、篡改、丢失的，应当立即采取补救措施，并通知履行个人信息保护职责的部门和个人；《网络安全法》同时要求网络运营者立即采取补救措施并按规定向有关主管部门报告。
- **网络安全事件**：依据《网络安全法》，发生网络安全事件时应当立即启动应急预案、采取补救措施，并按规定向有关主管部门报告；关键信息基础设施运营者还须履行更严格的报告与处置要求。
- **金融、电信等重点行业**：行业监管规定可能要求另行向行业主管部门报告，请一并核对该行业的最新规定。
- 报告渠道以各主管部门公布的官方入口为准（例如 CNCERT/CC 与 12377 举报中心）；具体时限与材料要求以最新法规和主管部门规定为准。

由于报告时限很短，请把责任人与联络路径预先写进第 6 节的事件响应流程。

---

## 参考

- 上游项目：[Autumn-27/ARTEX](https://github.com/Autumn-27/ARTEX)（AGPL-3.0）。本文档是该仓库中文版的防御资料。
- 上级 [README 的安全与滥用警示、使用范围与法律告知](../README.md)。
- 本版本面向中文用户。无论是否涉及个人信息，未经授权的探测在多数司法辖区都构成犯罪，请务必先取得书面授权并约定测试范围，然后再进行。
- 通用 Web 安全加固的标准参考：[OWASP Top 10](https://owasp.org/www-project-top-ten/)、[OWASP ASVS（应用安全验证标准）](https://owasp.org/www-project-application-security-verification-standard/)、[OWASP API Security Top 10](https://api-security.owasp.org/)。

> 本指南会为提升防御与检测能力而持续补充。欢迎通过仓库 issue 提出要增补的检测规则或加固项。
