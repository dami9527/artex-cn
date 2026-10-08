# ARTEX 检测规则（Suricata / 网络）

中文 · [English](README.en.md)

[Sigma 规则](../sigma/)的网络层搭档。这些 [Suricata](https://suricata.io) 签名覆盖在线路上可观测的
两类 ARTEX 产物，所有指标都依据本仓库源码中核实过的字符串或行为，而不是推测。主机·日志·SIEM 层位于
[`../sigma/`](../sigma/)，整体图景由防御指南（[中文](../../docs/defense-zh.md) · [English](../../docs/defense-en.md)）
说明。

## 规则：[`artex.rules`](artex.rules)

- **sid 1000001**：`ARTEX enrichment prober User-Agent`。User-Agent 以 `artex-enrich/` 开头的
  入站 HTTP `GET`（资产补全探测器 `enrich/enrich.go:233`）。这是单次请求的存在性指标。
  `classtype: attempted-recon`。
- **sid 1000002**：`ARTEX enrichment prober high-rate enumeration`。同一个 User-Agent 超过
  `detection_filter` 阈值，即**每来源 300 秒 30 个请求**。这是单次规则会漏掉的、机器
  速度的倾泻频率。对应 Sigma 关联规则 `artex_enrich_scan_velocity`。
  `classtype: attempted-recon`。
- **sid 1000003**：`ARTEX worker WebFetch User-Agent`。User-Agent 以 `norma/` 开头的入站
  HTTP 请求（norma SDK 的 WebFetch 工具 `github.com/Autumn-27/norma/tool/webfetch.go`）。该 UA 在 norma 所有版本
  （v0.1.0–v0.4.3，已核实）中都硬编码，而记录代理不修改请求头
  （`traffic/traffic.go`），因此会原样到达目标主机的线路。与补全探测器不同，
  它在**攻击阶段**（主动漏洞探测）触发。`classtype: attempted-recon`。

## 范围与如实说明：部署前请阅读

- **有两个 ARTEX User-Agent 可在网络上观测。** 补全探测器在侦察阶段发送
  `artex-enrich/1.0`（`enrich/enrich.go:233`），norma SDK 的 WebFetch 工具在攻击阶段发送
  `norma/0.4`（`github.com/Autumn-27/norma/tool/webfetch.go`）。记录代理（`traffic/traffic.go`）不修改请求头，
  因此两个 UA 都会到达目标线路。其它 worker 工具（作为 Bash 子进程的
  `curl`、`nmap` 等）使用自身的 User-Agent，请用通用扫描器签名与
  [`../sigma/`](../sigma/) 中基于行为的 SIEM 规则来检测。
- **User-Agent 只在明文下可见。** 它出现在明文 HTTP 流量中，或在终止 TLS 的代理·WAF 处
  被检查时。端到端 TLS 会把它加密，因此请把它部署在真正能看到 HTTP 请求缓冲区的位置。
- **静态 User-Agent 可以被操作者改掉**，因此没有它并不**不**意味着安全。
  持久的信号是行为，也就是速度与广度。sid 1000002（以及 Sigma 关联层）以速度作为
  基准的原因、纯 Web 多阶段检测需要按环境定制的基础规则的原因都在这里。
- **有意排除的内容。** 自更新 User-Agent `artex-selfupdate` 走 HTTPS 到 GitHub，因此
  在网络上不可观测（仅凭 TLS SNI 太常见，不足以报警）。审计控制标记是
  操作者侧的日志产物，而不是朝向目标的流量，请用
  [`../sigma/artex_guard_audit_framing.yml`](../sigma/artex_guard_audit_framing.yml) 检测。
  服务器端口 `:8787` 与记录代理 `127.0.0.1:8788`（`cmd/artex/main.go`）属于主机
  取证（`ss`·`netstat`），不是网络签名。

## 验证与测试

已用 Suricata 8 验证。加载测试不需要流量，随时可运行。

```sh
# 语法 + 引擎加载测试（预期："Configuration provided was successfully loaded"）
docker run --rm -v "$PWD/detections/suricata":/r -w /r jasonish/suricata:latest \
  suricata -T -S artex.rules -l /tmp --init-errors-fatal
```

`--init-errors-fatal` 会把能解析但初始化失败的规则也变成硬错误，使加载测试无法在被悄悄丢弃的
签名存在时通过。

要确认规则确实触发，可复现的回归测试在 [`../tests/suricata/`](../tests/suricata/)
中。它先运行同样的加载检查，再用 scapy 合成确定性抓包，然后在它之上运行 `suricata -r`
并断言告警数量。只需要 Docker。

```sh
detections/tests/suricata/run.sh
```

它断言 sid 1000001 每个探测恰好触发一次（35 个流的抓包中触发 35 次），sid 1000002 超过 300 秒 30 个的
阈值（Suricata 8.0.7 上 **5** 次告警，第 31~35 个流），并且把同一份抓包换成良性（benign）浏览器
User-Agent 后告警为 **0**，以此确认签名具有特异性。
参见 [`../tests/README.md`](../tests/README.md)。若想改为用自己的流量确认，可对本地
服务器抓取回环的 `curl -A 'artex-enrich/1.0'` 并读取告警。

```sh
suricata -r enrich.pcap -S artex.rules -l out && \
  grep -c '"signature_id":1000001' out/eve.json    # 存在：每个探测一次
```

## 贡献

欢迎贡献检测规则。新规则应让每个指标都依据可观测的事实，在注释中
写明局限，干净地通过 `suricata -T`，并且不含任何读起来像攻击指引的内容。
参见 [`../../CONTRIBUTING.md`](../../CONTRIBUTING.md) 与 [`../sigma/`](../sigma/) 的 Sigma 层 /
[`../README.md`](../README.md)。
