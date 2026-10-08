# 贡献指南（Contributing）

中文 · [English](CONTRIBUTING.en.md)

感谢你对 ARTEX 中文版（`artex-cn`）的关注。本文用简体中文说明在开始贡献之前需要了解的范围、
方针与流程。提交贡献之前，请务必先读[使用范围与法律责任](#使用范围与法律责任)与
[本地化方针](#本地化方针)。

- 要报告缺陷或提出功能建议 → 使用[议题模板](https://github.com/dami9527/artex-cn/issues/new/choose)。
- 发现翻译或本地化错误 → 请使用「翻译·本地化错误」议题模板。
- 发现安全漏洞 → **不要提交公开议题**，请按 [SECURITY.md](SECURITY.md) 的流程处理。
- 所有参与者都必须遵守[行为准则（CODE_OF_CONDUCT.md）](CODE_OF_CONDUCT.md)。

---

## 使用范围与法律责任

ARTEX 是让 LLM 多智能体**自主**执行渗透测试的攻击安全工具。贡献者与使用者受同样的范围限制。

- 验证代码时，只能对**自己所有，或已获得书面明确授权的目标**，或**本地隔离环境**
  （例如用 Docker 起的 OWASP Juice Shop、DVWA 这类故意留有漏洞且属于本人的目标）
  运行本工具。
- 不接受对超出授权范围的真实、生产、远程系统执行扫描、探测、利用的代码，也不接受
  鼓励此类用法的改动。
- 在中国境内，未经授权侵入他人信息网络、干扰其正常运行，违反《中华人民共和国网络安全法》；
  过程中收集、接触到的个人信息受《中华人民共和国个人信息保护法》约束，涉及重要数据的
  还适用《中华人民共和国数据安全法》。完整告知见 [README](README.md#️-请先阅读--使用范围与法律告知)。

贡献所提交的代码、文档被如何使用，其法律责任由实际运行它的使用者本人承担。本仓库按
「现状（AS IS）」提供。

---

## 本地化方针

本仓库存在的意义，是在保留上游 [Autumn-27/ARTEX](https://github.com/Autumn-27/ARTEX)
**判断性能**的前提下，把面向用户的产出统一为**简体中文**。偏离这一方针的改动可能降低性能，
因此不予接受。

- **不要翻译 agent 的内部推理提示词（行为指令正文）。** 必须保留按原文（中文）基准测试过的
  行为。该正文位于 `agent/promptcatalog.go` 与数据库种子（`agent_prompts`）。翻译会让 agent
  的判断发生漂移。
- **只强制面向用户的产出使用简体中文。** 检测结果（`report_finding`）、事实摘要
  （`record_fact`）、最终报告、对话回复都属于这一类。这条强制由 `agent/prompt.go` 中名为
  `langDirective()` 的代码固定尾段追加到各角色 system 提示词末尾。要改输出语言就改这个函数。
- **命令、payload、代码、URL、日志原文不翻译。** 它们是分析所需的原始材料，保持原样。
- **保留上游中文原文。** 文档把原文留在 `README.zh.md`，便于与上游（upstream）仓库的变更
  对照。界面文案集中在 `web/messages/zh.json`。
- UI 文案不要硬编码，按消息文件的键添加。
- **仓库里不允许出现韩文。** 本仓库曾经是韩文版，界面、注释、后端文案、提示词、文档、脚本
  与入库数据都被翻译过一遍。为避免回退，仓库把这条约束固化成了可执行检查
  [`scripts/check-no-korean.py`](scripts/check-no-korean.py)：它扫描工作树下的文本文件
  （跳过 `.git`、`node_modules` 等依赖目录、构建产物与二进制文件），
  只要任何一行还有谚文就失败。提交前请运行 `python3 -I scripts/check-no-korean.py`；
  需要修改检测范围时改脚本本身（不要用豁免清单绕过）。提交信息同样一律用简体中文书写。
- **输出语言强制是提示词引导，而不是硬性固定。** `langDirective()` 只是**指示**输出语言为
  简体中文，并不强制固定。所以中文的忠实度取决于模型能力、角色与上下文。验证本地化改动时，
  请使用能力足够的模型（前沿级）。低价、小模型可能让报告与摘要退回原文，因此不要只凭低价模型
  的输出判断本地化是否生效。使用 OpenAI 系列模型时 `max_tokens` 的设置陷阱整理在
  [README 的「模型选择与输出语言」一节](README.md#模型选择与输出语言)。
- **追赶上游（upstream）变更的流程写在维护者指南里。** 上游 ARTEX 更新后，如何区分要保留的
  资产与要翻译的内容并落地，以及如何检查翻译对称与翻译漂移，整理在
  [MAINTAINING.md](MAINTAINING.md) 中。

---

## 开发环境

本项目由 **Go 后端**（单个二进制内嵌前端）+ **Next.js 前端**组成。

主要功能的设计意图整理在 `docs/` 的设计文档里。涉及把漏洞与流量证据关联起来的功能
（报告 agent 的自动绑定、`report_finding` 的 `traffic_refs` 等）时，请先阅读
[漏洞多流量证据设计文档](docs/finding-traffic-evidence-zh.md)。

### 所需版本

- Go 1.26 或更高（以 `go.mod` 为准）
- Node.js 22 或更高（以发布工作流为准）
- Docker 与 Docker Compose（本地运行与验证用）

### 后端（Go）

本机装了 Go 时，在仓库根目录执行以下命令。

```bash
go build ./...
go vet ./agent/
go test ./agent/
```

本机没有 Go 时，可以用 Docker 做同样的验证。把模块缓存与构建缓存放在 named volume 上，
重复执行会更快。

```bash
docker run --rm -v "$PWD":/src -w /src \
  -v artexcn-gomod:/go/pkg/mod -v artexcn-gocache:/root/.cache/go-build \
  golang:1.26 sh -c 'go build ./... && go vet ./agent/ && go test ./agent/'
```

#### 数据库集成测试（需要 postgres）

上面的 `go test ./agent/` 会**悄悄跳过**必须连上 PostgreSQL 才能跑的**数据库集成测试。**
`agent`、`config`、`db`、`evidence`、`llmrec`、`server` 这六个包里都有需要真实数据库才能跑的
测试：既没有环境变量 `ARTEX_PG_DSN`，配置文件里也没有 `database` 项时，这些测试会以
`--- SKIP` 跳过，整个包以 `ok` 结束。所以改完这六个包后在无 DSN 的情况下验证，可能
**本地通过（ok），而 PR 的 `go-db` 作业失败**。

要在本地跑这些测试，需要起 PostgreSQL 并把 `ARTEX_PG_DSN` 传进去。下面是拉起与 CI 相同的
`postgres:16-alpine`、放在隔离网络里运行的例子，复用了上面的同名 named volume。

```bash
# 1) 拉起隔离网络与空的 postgres（镜像与账号和 CI 相同）。
docker network create artexcn-db 2>/dev/null || true
docker run -d --name artexcn-pg --network artexcn-db \
  -e POSTGRES_USER=artex -e POSTGRES_PASSWORD=artex -e POSTGRES_DB=artex \
  postgres:16-alpine
until docker exec artexcn-pg pg_isready -U artex -d artex >/dev/null 2>&1; do sleep 1; done

# 2) 传入 DSN，跑数据库集成包（DSN 里的 host 就是容器名）。
#    只跑改过的包时，把 ./agent/ 换成 config、db、evidence、llmrec、server。
docker run --rm --network artexcn-db -v "$PWD":/src -w /src \
  -v artexcn-gomod:/go/pkg/mod -v artexcn-gocache:/root/.cache/go-build \
  -e ARTEX_PG_DSN='postgres://artex:artex@artexcn-pg:5432/artex?sslmode=disable' \
  golang:1.26 sh -c 'go test ./agent/ -count=1'

# 3) 清理。
docker rm -f artexcn-pg && docker network rm artexcn-db
```

CI 的 `go-db` 作业会在合并前强制跑这六个包，且**每个包各自用独立的 postgres 隔离**
（`.github/workflows/ci.yml`）。改过数据库集成包时，建议在提 PR 之前用上面的方法把对应包
亲自确认一遍。

### 前端（web）

```bash
cd web
npm ci
npm run dev          # 开发服务器
npm run build        # 生产构建
npm run build:static # 静态导出构建（含合并门禁与 TypeScript 类型检查）
npm run check        # Biome 检查与格式化（已是合并门禁，必须 0 错误）
npm run check:fix    # 自动修复
```

提交前的格式化与检查由 Biome 管理。`lint-staged` 会对暂存的文件自动执行
`biome check --write`。

### 整体运行（Docker Compose）

```bash
cp .env.example .env     # 设置 POSTGRES_PASSWORD
docker compose up -d     # 启动 artex + postgres → http://localhost:8787
```

---

## 贡献流程

1. 先**开议题。** 较大的改动最好在动手之前用议题对齐方向。小修小补（错别字、链接、
   明显的缺陷）可以直接提 PR。
2. **Fork** 仓库并建主题分支。分支名把改动性质放在前面，如 `feat/...`、`fix/...`、
   `docs/...`、`i18n/...`。
3. 写完改动后**亲自跑对应范围的验证。** Go 改动要让上面的
   `build`、`vet`、`test` 通过。web 改动要让 `npm run build:static`
   （同时执行合并门禁与 TypeScript 类型检查）通过。
   `npm run check`（Biome）也是合并门禁：`web.yml` 里这一步名为
   `biome check（合并门禁 · lint、format、a11y 保持 0 错误）`，一报错整个 job 就失败，
   所以必须让全部检查通过、保持 0 错误（提交时 `lint-staged` 只对暂存的文件自动应用
   `biome check --write`，可以先用它把格式问题修掉再复查）。
   改过文档（`.md`）时，用 `python3 -I scripts/check-doc-links.py` 确认仓库内的
   链接、图片引用与文档锚点（`#标题`）链接没有断。锚点按与 GitHub 相同的规则
   从标题生成 slug 再比对，所以改了标题文字却没改指向该标题的锚点链接，就会在
   这里被拦住（CI 的 `docs` 工作流把同一条检查作为合并门禁强制执行）。这条检查
   也作为 `docs` 钩子写进了仓库根目录的
   [`.pre-commit-config.yaml`](.pre-commit-config.yaml)，只要执行过
   `pre-commit install`，提交时就会自动运行（它只用 Python 标准库、不联网，
   所以没有 Docker 也能跑完）。
4. **提 PR。** 标题与描述遵循 [PR 模板](.github/PULL_REQUEST_TEMPLATE.md)，
   写清改了什么、为什么改、怎么验证的。改过 UI 就附上截图。
5. 属于用户可见的改动（功能、本地化、文档、检测规则等）时，在
   [变更历史（CHANGELOG.md）](CHANGELOG.md) 的 `[Unreleased]` 一节加一行。
   内部重构或只改测试的改动可以省略。

### 提交信息

沿用既有提交历史的惯例。格式为 `type(scope): 说明`，说明一律用简体中文书写。

- `type`：`feat` · `fix` · `docs` · `chore` · `refactor` · `test` · `i18n` 等
- `scope`：改动的区域（`agent` · `web` · `server` 等），可省略

示例如下。

```
feat(agent): 强制用户可见输出为简体中文（langDirective）
docs: 撰写中文 README，原文保留在 README.zh.md
i18n(web): 仪表盘导航标签中文化
```

---

## 检测规则与检测测试贡献

本仓库把用于**防御与检测** ARTEX 这类自主 AI 攻击的规则一并放在 [`detections/`](detections/)：
可部署的 [Sigma](https://sigmahq.io) 规则（[`detections/sigma/`](detections/sigma/)）、面向网络的
[Suricata](https://suricata.io) 规则（[`detections/suricata/`](detections/suricata/)）、
[MITRE ATT&CK](https://attack.mitre.org/) 覆盖率图层（[`detections/attack/`](detections/attack/)），
以及可复现地证明这些规则确实会触发的测试（[`detections/tests/`](detections/tests/)）。新提交或修改
检测规则时请遵守下面的契约。八套测试会机械地强制其中大部分要求，所以只改规则而不更新测试与图层，
测试就会失败。

- **所有指标都要落在可观测的事实上。** 规则里使用的字符串、User-Agent、行为阈值必须能在本仓库
  源码中实际查到，不能凭猜测编造。请在规则内注明依据的源文件（例如 `artex-enrich/1.0` 指标可以在
  `enrich/enrich.go` 中查到）。指标一致性测试
  （[`detections/tests/indicators/`](detections/tests/indicators/)）会检查每个指标是否仍同时存在于上游
  源码与规则中，所以上游重新同步改了源码字符串时，不同步修改规则就会测试失败。
  改动机器可读的指标清单（[`detections/indicators/artex_indicators.csv`](detections/indicators/artex_indicators.csv)）时，
  也要一并更新原样收录这些指标的 MISP 事件
  （[`detections/indicators/artex_indicators.misp.json`](detections/indicators/artex_indicators.misp.json)）。
  MISP 导出测试（[`detections/tests/misp/`](detections/tests/misp/)）会强制两个文件逐行一致，
  并确认该事件是能被 pymisp 载入的合法 MISP 文档。
- **如实写明局限。** Sigma 规则在 `description` 里、Suricata 规则在注释里写明该规则抓不到的情形
  与误报可能。若某条只是通用狩猎线索（例如破坏性命令）而非 ARTEX 独有特征，就要明确说明，
  避免仅凭一次命中就把攻击者断定为 ARTEX。
- **通过静态校验。** Sigma 规则必须让 SigmaHQ 校验器基准以零问题通过
  （`sigma check --validation-config detections/tests/sigma_lint/validators.yml`）。默认的
  `sigma check` 只跑 pySigma 核心校验器，标题写法、字段/日志源分类、参考链接这类 SigmaHQ 惯例
  只有用这份基准才能筛出来。有四处例外属于不适合单规则集的 SigmaHQ 单仓惯例，理由已写在
  [`detections/tests/sigma_lint/validators.yml`](detections/tests/sigma_lint/validators.yml) 里。
  Suricata 规则必须能被 `suricata -T` 干净加载。
- **一并提交可复现的测试。** 用 [`detections/tests/`](detections/tests/) 下的测试证明规则会触发
  （或结构合法）。输入不要以二进制形式放进仓库，每次确定性生成；与引擎版本无关的属性
  （是否触发、有无误报）要精确断言，随版本波动的数值断言其下界并单独记录基准值。新增或修改 Sigma
  关联规则时，后端可移植性测试（[`detections/tests/sigma_backends/`](detections/tests/sigma_backends/)）
  会检查该规则能否在多个后端上转换，因此请保持与
  [`detections/README.md`](detections/README.md) 中后端支持说明一致。
- **同步更新 ATT&CK 图层。** 给规则增删或修改 `attack.*` 标签时，
  [`detections/attack/artex_navigator_layer.json`](detections/attack/artex_navigator_layer.json) 中的
  技法与分数也要跟着改。一致性测试会强制规则与图层双向吻合，所以图层里没有的规则标签、
  或规则里没有的图层技法都会导致失败。
- **不要写读起来像攻击教程的内容。** 本仓库的检测资料只保持防御与检测的定位。不接受讲解如何
  实施利用、如何规避检测之类有助于攻击的描述。

八套测试只要有 Docker 就能直接跑，产物不提交进仓库。每个脚本只要有任一断言失败就以非零码结束，
因此可以直接接入 CI 或 pre-commit 钩子。

```bash
detections/tests/sigma/run.sh           # Sigma：sigma check + 后端转换 + 指标保留
detections/tests/sigma_match/run.sh     # Sigma：原子规则对恶意样本触发、对正常样本静默
detections/tests/sigma_lint/run.sh      # Sigma：SigmaHQ 惯例全量校验器 + 已记录基准
detections/tests/sigma_backends/run.sh  # Sigma 可移植性：关联规则能否在多个后端转换
detections/tests/suricata/run.sh        # Suricata：合成 pcap → suricata -r → 断言告警数
detections/tests/attack/run.sh          # ATT&CK：图层 ↔ 规则双向一致
detections/tests/indicators/run.sh      # 指标：规则固定的指标 ↔ 上游源码双向一致
detections/tests/misp/run.sh            # MISP：指标 CSV ↔ MISP 事件同步 + pymisp 校验
```

要一次跑完八套，使用 [`detections/tests/run-all.sh`](detections/tests/run-all.sh)。它按与 CI 相同的
顺序依次执行八套，前面的套件失败也会把其余套件跑完，最后输出各套件的 PASS/FAIL 汇总，只要有
一个失败就以非零码结束。把这个运行器直接挂成 pre-commit 钩子的配置示例在仓库根目录的
[`.pre-commit-config.yaml`](.pre-commit-config.yaml) 里。执行 `pip install pre-commit &&
pre-commit install` 安装后，只有在检测规则或规则所固定的上游源码发生改动的提交里（范围与 CI 相同）
运行器才会跑起来，把规则与测试不一致的问题挡在推送之前。同一个配置文件里还包含检查文档内部链接、
图片与锚点的 `docs` 钩子（上面贡献流程第 3 步的 `check-doc-links.py`），以及扫描全仓库韩文残留的
`ko-check` 钩子（上面「不允许出现韩文」一节的 `check-no-korean.py`，每次提交都会跑）。

这八套测试由仓库 CI（[`.github/workflows/detections.yml`](.github/workflows/detections.yml)）在
`detections/` 下发生改动的每次推送与 PR 上运行。指标一致性测试还会在该指标所指向的上游源文件
（`enrich/`、`selfupdate/`、`guard/`、`db/`、`cmd/artex/main.go`）发生改动时运行，一并抓住上游重新
同步改了 User-Agent、标记或默认端口而导致规则悄悄失效的情况。因此，只改规则而不更新测试与图层、
破坏 SigmaHQ 惯例，或与源码脱节的规则，都会在合并前由 CI 明确暴露出来。

规则索引以及每条规则的依据与局限见 [`detections/README.md`](detections/README.md)，测试的断言项与
运行方法见 [`detections/tests/README.md`](detections/tests/README.md)。

---

## 许可证

本项目采用 **GNU Affero General Public License v3.0（AGPL-3.0）** 授权。
提交贡献即视为你**同意该贡献同样以 AGPL-3.0 公开**。
特别地，若你修改本项目并通过网络（例如作为在线服务）提供给用户，就必须向这些用户公开
对应的完整源码。完整条款见 [LICENSE](LICENSE) 文件。
