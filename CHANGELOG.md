# 变更历史

中文 · [English](CHANGELOG.en.md) · [中文（上游原文）](CHANGELOG.zh.md)

本文记录 ARTEX 中文版（本 fork）相对上游仓库所做的变更。格式参考 [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)。

上游 ARTEX 项目的逐版本发布历史（0.3.x 及更早）与贡献者名单，按上游中文原文保留在 [`CHANGELOG.zh.md`](CHANGELOG.zh.md) 中。为便于与上游变更对照，与 `README.zh.md` 采用同样的方式保留原文。各项变更的细节与依据可在仓库提交历史中查看。

## [Unreleased] · 中文版变更

### Docker 部署路径修复

- **让 Docker 能起中文版了。** `docker-compose.yml` 的 `artex` 服务原本拉取 `autumn27/artex` 镜像，而该镜像已随原作者关闭仓库从 Docker Hub 消失（现在 pull 会返回 `not found`，`autumn27` 命名空间下只剩两个 `scopesentry`）。于是在 compose 里加上 `build:`，把镜像名改成本地标签（`${ARTEX_IMAGE:-artex-cn:local}`）：**在已备好 `dist/<arch>/artex` 的前提下（即已跑过 `./build-image.sh`）**，一行 `docker compose up -d --build` 就能直接构建并启动中文版镜像；全新 clone 里没有这个二进制，直接跑 compose 会停在 `COPY dist/<arch>/artex` 处失败。原来走上游镜像的路径（`docker compose pull`）已不再成立。
- **把 `Dockerfile` 是「只负责运行」这一前提落到了文档与脚本里。** 这个 Dockerfile 不在容器内编译，而是用 `COPY dist/<arch>/artex` 放入事先编译好的 Linux 二进制。因此需要 ① 前端静态构建 → ② `GOOS=linux` 交叉编译 → ③ `docker build` 的顺序，跳过前面的步骤就会在 `COPY` 处失败。README 新增了[「自行构建中文版镜像」](README.md#自行构建中文版镜像)一节，写明这个顺序、`--platform` 与 `GOARCH` 需要一致，以及 `build.sh` 与 Dockerfile 期望的路径不同（`dist/artex-linux-amd64/` vs `dist/<arch>/`）。
- **修好了 `install.sh` 的「① 全部 Docker」与 `update.sh` 的 Docker 路径拉取已消失镜像的问题。** 现在两个脚本都用当前源码重新构建镜像（`docker compose up -d --build`）。缺少所需工具（Go、Node.js/npm、rsync）时会就地提示并停下。`update.sh` 里已不再使用的 `ARTEX_TAG` 输入步骤也已删除。
- **把 `.env.example` 与实际行为对齐。** 用 `ARTEX_IMAGE`（本地构建出的镜像标签）替换 `ARTEX_TAG`（上游镜像标签），并明确本仓库不发布镜像。
- **记录下改密码后导致启动失败的问题。** PostgreSQL 官方镜像只在**数据卷为空时读取一次** `POSTGRES_PASSWORD` 并写入 `initdb`，所以在栈已经起过一次之后再改 `.env` 里的密码，这个新值会被忽略，只有 artex 用新密码去连接，陷入 `28P01` 认证失败 → `异常退出（code=1）` 的重启循环（端口虽被 Docker 占着，容器里却没有监听进程，浏览器看到的是连接失败）。README（zh·en）新增[「页面打不开时（Docker）」](README.md#页面打不开时docker)一节，写明查看日志的方法与两种恢复方式（`down -v` 后重新初始化 / 用 `ALTER USER` 保留卷内数据对齐密码），`.env.example` 里也加了同样的警告。顺序始终是 **先 `.env` → 再 `docker compose up -d --build`**。

### 界面语言（简体中文）

- **服务端播种的 agent 显示名也跟随运行时语言了。** 曾经出现界面与播种名不一致：界面按简体中文构建，对话页的 agent 选择器里 reporter 却仍留着另一种语言的历史遗留名称（用户反馈）。原因是这个名称**不是前端文案，而是数据库播种数据**——`reporter`、`retester` 的名称与描述在播种时以字符串写入数据库（分别由 `reporter_agent_seed_v1`、`finding_retester_seed_v1` 标记只执行一次），而 Web UI 的 locale 在静态导出时就写进了 HTML，服务端无从得知。
  - 新增运行时配置 `ARTEX_LOCALE`（取值只有 `zh`，未设置或不支持的值都回落到 `zh`），用它对齐播种标签。部署脚本 `build-image.sh` 会把构建时选定的语言自动传进容器，因此界面与服务端不会错位（`.env` 里写一次即可）。
  - 为**已经安装好的实例**补了 `localizeSeedAgentNames`。只有当当前值落在该 agent 的**历代默认值集合**（上游中文原文与历史本地化版本，含文案变更过的历史值）里时——也就是**用户没有在界面里改过**时——才对齐到新的 locale 值，用户改过的名字保持不变。用 `seed_agent_names_locale` 标记实现幂等，并且**标记已经是当前 locale、实际状态却对不上时仍会重新对齐**。
  - 实现过程中自己抓出了两个缺陷：① 判断「已经是目标值就跳过」时拿目标值去比较，导致目标值恰好等于另一项的默认值（历史遗留中文）时，后面的项被整段跳过。② 这个缺陷没能改掉名字却留下了标记，后续启动据此认为「已经完成」，于是永远修不好。现在用**历代默认值集合匹配 + 状态复查**，两个问题都不存在了。这两个缺陷由真正连数据库的集成测试（`TestLocalizeSeedAgentNames`，四个子用例：把历史描述对齐到当前中文标签、保留用户改过的名字与描述、标志已是当前 locale 但状态不一致时重新对齐、标志与状态都一致时不做任何改动）固定下来——只看常量的静态测试抓不到这一类问题。本仓库固定为中文，不再有跨语言切换，因此这里只断言最终名称与描述是中文且不含谚文。
- **把仍以英文残留的界面文案改成了中文。** 这是用户反馈后找到的两处遗漏。
  - **侧边栏「环境设置（Preferences）」面板**（`web/src/app/(main)/_components/sidebar/layout-controls.tsx`）：只有这个文件完全没有用 `next-intl`，标题、说明、8 个标签、12 个开关与 `aria-label` 全是硬编码英文（主题预设、字体、主题模式、页面布局、顶栏行为、侧边栏样式、折叠方式、恢复默认值）。新增 `layoutControls` 命名空间，全部改用 `t()`。主题预设的 `label`（theme.ts）是英文标识符，值保持不变，只映射显示名称；字体名（Geist、Roboto 等）是专有名词，不翻译。
  - **404 页面**：导出的 `404.html` 是 Next 默认的英文页面（`Page not found.`）。原因有两点——① App Router 的 `not-found.tsx` 只生成供 `notFound()` 使用的内部路由，不会用于静态导出的顶层 404（Next 16 需要 `global-not-found.tsx` + `experimental.globalNotFound`）。② `zh.json` 的 `notFound` 三个键仍是**英文值**（从上游中文原文迁移时遗漏）。于是新增 `global-not-found.tsx`（直接返回完整 HTML 文档）及其正文组件，并把 `zh.json` 的 `notFound`、`search.empty` 填成中文。404 不继承根布局，所以要把消息直接传给 `NextIntlClientProvider`（不传的话静态导出时解析不出文案，又会输出英文页面）。
- **新增 `scripts/check-no-korean.py`，把「不允许韩文残留」变成可执行的门禁。** 像上面 ② 那样遗留下来的**非中文值**，靠人工比对是抓不到的，只会悄悄漏出去。这个脚本扫描工作树下的文本文件（跳过 `.git`、`node_modules` 等依赖目录、构建产物与二进制文件），逐行匹配谚文，只要命中就以非零码失败并列出文件、行号与命中内容。它看的是源码而不是构建产物，因此与构建语言无关。
- **新增 `build-image.sh`，把界面语言切换做成一条命令。** 之前的文档与脚本只做到「让构建路径把 `NEXT_PUBLIC_LOCALE` 传下去」，却漏掉了**在 `docker compose up -d --build` 前面加这个变量并不会改变语言**这一事实。本项目的 `Dockerfile` 不在容器里编译前端（用 `COPY dist/<arch>/artex` 放入事先构建好的二进制），而那份二进制里 embed 了 `web/out`，所以 compose 只会把这次拷贝以 `CACHED` 结束。也就是说界面语言是**在宿主机上跑 `next build` 时**定下来的。`build-image.sh` 代替人工执行 ① 前端 → ② 同步内嵌目录 → ③ `GOOS=linux` 编译 → ④ `docker compose up -d --build`，并把最后一次构建的语言记在 `dist/<arch>/.locale` 里，**只在语言变化时重新构建**（相同则复用，`--force` 强制重建，`--no-up` 跳过启动）。用 `./build-image.sh` 即可。
- **把 `install.sh`、`update.sh` 的构建逻辑委托给 `build-image.sh`。** 同一套逻辑写两份会让界面语言悄悄错位（实际就错位过），所以集中到一处，两个脚本只负责调用。
- **用 `NEXT_PUBLIC_LOCALE` 选择界面语言。** `web/src/i18n/config.ts` 里本来就有 `LOCALES`、`DEFAULT_LOCALE`、`resolveLocale()`，但**构建路径里任何地方都没有传 `NEXT_PUBLIC_LOCALE`**，所以做不出中文界面（总是落到默认值）。locale 在静态导出时就写进 HTML，因此这不是运行时切换，而是构建期选择。本仓库最终把界面固定为简体中文：`LOCALES = ["zh"]`、`DEFAULT_LOCALE = "zh"`，取值不是 `zh` 时回落到 `zh`。
- **`scripts/check-no-korean.py` 的判定范围与跳过规则。** 它只查谚文——谚文音节、谚文字母、谚文兼容字母、谚文扩展区 B，不查汉字也不查假名：中文文案本来就全是汉字，查汉字等于自我误报；假名属于日文，不在清理目标内。同时跳过版本控制与依赖目录（`.git`、`node_modules`、`.next`、`out`、`dist`）、构建产物（`server/webui/dist`、`web/out`）与二进制文件（无法按 UTF-8 解码的一律跳过）。需要豁免的文件写进脚本的忽略清单，当前为空：仓库里不应该有任何需要豁免的文件。
- **在文档里写明：变化的只有界面文案。** agent 产出的漏洞报告、事实摘要、最终总结与聊天回复的语言由 Go 代码（`agent/prompt.go` 的 `langDirective()`）固定为简体中文，与上面的值无关。要改输出语言，就得同时改 `langDirective()` 与验证该契约的 `agent/prompt_test.go`，而这等于回退本 fork 的中文化设计。

### 上游同步（v0.3.15）

把 fork 点（上游 `d003372`，2026-10-03）之后上游新增的三个提交带进了本仓库。三者都是 fork 之后的上游变更，所以合并冲突只局限在本仓库已经改过的文件里，这些冲突点按下文所述的中文版约定解决。

- **单独计量模型回退审批的 token 用量（[`db/llm_usage.go`](db/llm_usage.go) · `server/intercept.go`）。** 审批调用此前没有独立的计量归属，看不出花了多少。现在用 `worker=judge` 单独统计，并在「系统 → 拦截」的「模型回退审批」开关下方放一张用量卡片（调用次数、输入、输出、缓存读、缓存写，最近 30 天按日柱状图）。上游把这张卡片的文案硬编码成了中文；本仓库把新文案抽成 `web/messages/zh.json` 的 `interceptPage.judgeUsage.*` 键，走与既有界面相同的 i18n 路径。
- **堵住了认证初始化的 fail-open（上游 `a951e4a`，`server/auth.go` · `db/settings.go`）。** 读取密码相关配置失败时若当成「尚未设置」处理，未认证的请求就能覆盖已经设置好的管理员密码。现在读失败返回 503，把「只能设置一次」的保证放在主键约束（`INSERT ... ON CONFLICT DO NOTHING`）上而不是 upsert，并在服务端强制密码长度校验（8 位以上，bcrypt 上限 72 字节）。中文版把新的错误文案与校验消息译成中文，并把 setup 界面的「无法确认」状态文案加为 `auth.setup.*` 键。
- **把上游 `v0.3.15` 的变更历史并入了 [`CHANGELOG.zh.md`](CHANGELOG.zh.md)。** 上游的定版提交与本仓库分离的变更历史结构不匹配，所以没有直接 cherry-pick，而是把 `[Unreleased]` 里的内容整理成 `## [0.3.15] - 2026-10-07` 的方式搬运原文。

### 本地化（i18n）

- **强制用户可见输出为简体中文。** 经过基准测试的 agent 行为指令正文（大脑）为保留性能而原样保留，用代码固定段（`langDirective`）要求只把用户可见的产出（漏洞报告、事实摘要、最终总结、聊天回复）写成简体中文。命令、payload、代码、日志原文保持原样。
- **把 Web UI 中文化。** 在 Next App Router 里引入 `next-intl`，把字符串集中到 `web/messages/zh.json`。仪表盘、漏洞、对话、通知发送、拦截、LLM 设置等界面文案都改成了简体中文。
- **把服务端 API 的用户可见错误与响应中文化。** 返回浏览器的 HTTP 错误与响应文案改成简体中文。但回喂给 agent 大脑的文案为防基准漂移而保留原文，判定依据记录在仓库工作文档里。
- **整理中文文档。** 撰写简体中文 `README.md`，同时保留英文 `README.en.md`，上游中文原文保留在 `README.zh.md`。

### 防御与检测资料

- **新增防御与检测指南。** 整理了自主 AI 攻击与既有扫描器的差异、防御者可观测的指纹（IoC）、入口与加固、检测规则、事件响应的中文指南（[`docs/defense-zh.md`](docs/defense-zh.md)），以及内容相同的英文版（[`docs/defense-en.md`](docs/defense-en.md)）。
- **提供可部署的检测规则。** 把指南里的指纹变成可直接使用的规则：主机与日志层用 [Sigma](https://sigmahq.io) 原子规则与关联规则（[`detections/sigma/`](detections/sigma/)），网络层用针对探测器 User-Agent 与 norma SDK WebFetch User-Agent 的 [Suricata](https://suricata.io) 规则（[`detections/suricata/`](detections/suricata/)）。
- **把 ATT&CK 覆盖率可视化。** 将规则标注的技法整理成 MITRE ATT&CK Navigator 图层（[`detections/attack/`](detections/attack/)）。
- **以标准格式提供机器可读的入侵指标（IoC）。** 把 ARTEX 输出的独有指纹汇总成一个 CSV（[`detections/indicators/artex_indicators.csv`](detections/indicators/artex_indicators.csv)），并做成可直接导入威胁情报平台的 MISP 事件（[`detections/indicators/artex_indicators.misp.json`](detections/indicators/artex_indicators.misp.json)）。有规则支撑的指标标为 `to_ids`，主机取证端口则作为分类线索区分标注。
- **附上可复现的检测测试。** 加入八种真正跑规则来证明的测试（Sigma 结构与编译校验、Sigma 实时事件匹配、后端可移植性、SigmaHQ 惯例检查、Suricata 加载与触发、ATT&CK 图层一致性、指标与源码一致性、MISP 导出 ↔ CSV 同步），以及一次跑完它们的批处理运行器与 pre-commit 示例，并接入 CI 合并门禁。Sigma 实时事件匹配不仅确认规则能编译，还要确认它对恶意样本事件真的触发、对正常事件保持静默。

### 仓库整理

- **加入安全与滥用警告以及合规告知。** 在 README 顶部加入使用范围、《网络安全法》《数据安全法》《个人信息保护法》的告知与禁止滥用警告。
- **把界面预览截图换成中文界面截图。**
- **整理维护者手册与贡献指南。** 放入防止上游同步漂移与翻译漂移的手册（[`MAINTAINING.md`](MAINTAINING.md)）与检测规则贡献契约（[`CONTRIBUTING.md`](CONTRIBUTING.md)）。手册里还整理了发布流水线的构建前提，以及不打标签在本地验证这些前提的流程。
- **加入推送与 PR 的合并门禁 CI。** 上游仓库只在打标签发布时跑 CI，而本 fork 在每次推送与 PR 上运行 Go 构建、静态分析（`go vet`）、单元测试（[`ci.yml`](.github/workflows/ci.yml)）、中文界面静态构建（[`web.yml`](.github/workflows/web.yml)）与文档内部链接及图片引用的完整性检查（[`docs.yml`](.github/workflows/docs.yml)），把中文化过程中产生的回归挡在合并之前。需要数据库的集成测试按包用隔离的 PostgreSQL 服务一并验证。文档链接检查用不依赖外部网络的确定性脚本（[`scripts/check-doc-links.py`](scripts/check-doc-links.py)）运行，防止多语言文档之间大量相对链接与界面预览图在断开的状态下被合并。文档锚点（`#标题`）链接也按与 GitHub 相同的 slug 规则与标题比对，一并抓住因标题改动而悄悄失效的目录与交叉引用。检测规则套件由上面「防御与检测资料」一节说明的合并门禁负责。
- **定期检查外部链接存活。** 防御指南指向的事故报告渠道、标准参考等外部链接依赖远端服务器状态、容易 flaky，所以从合并门禁中拿掉，改用非阻塞工作流（[`external-links`](.github/workflows/external-links.yml)）在每周一与手动触发时按浏览器 User-Agent、GET、跟随重定向的方式检查（[`scripts/check-external-links.py`](scripts/check-external-links.py)）。主机活着、只是确认方式被挡（机器人拦截、速率限制）以及我们修不了的上游继承死链（allowlist）都不算失败，因此只有我们文档收录的外部链接新出现失效时才会报红。
- **搭好贡献与治理基础设施。** 放入缺陷、功能、翻译议题模板（[`.github/ISSUE_TEMPLATE/`](.github/ISSUE_TEMPLATE/)）与 Pull Request 模板（[`PULL_REQUEST_TEMPLATE.md`](.github/PULL_REQUEST_TEMPLATE.md)）、安全漏洞报告政策（[`SECURITY.md`](SECURITY.md)）、行为准则（[`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md)），让外部贡献者以一致的格式提交议题、PR 与安全报告。
- **补齐面向海外贡献者的英文文档层。** 本仓库以简体中文为主语言，但为了让读不懂中文的贡献者、安全研究者与防御者也能获得同样的信息，关键文档一并提供英文版：英文 `README.en.md`、防御指南（[`docs/defense-en.md`](docs/defense-en.md)），以及变更历史（[`CHANGELOG.en.md`](CHANGELOG.en.md)）、安全报告政策（[`SECURITY.en.md`](SECURITY.en.md)）、行为准则（[`CODE_OF_CONDUCT.en.md`](CODE_OF_CONDUCT.en.md)）、贡献指南（[`CONTRIBUTING.en.md`](CONTRIBUTING.en.md)）、维护者手册（[`MAINTAINING.en.md`](MAINTAINING.en.md)）、流量证据设计文档（[`docs/finding-traffic-evidence-en.md`](docs/finding-traffic-evidence-en.md)）与缺陷、功能、翻译议题模板的英文版。中文版与英文版在开头互相指路，从任一语言进入都能切到另一边。（Pull Request 模板目前只提供中文版。）

---

上游 ARTEX 项目的逐版本发布历史与贡献者名单，可在 [`CHANGELOG.zh.md`](CHANGELOG.zh.md) 中原样查看。
