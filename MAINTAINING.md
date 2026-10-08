# 上游同步与翻译漂移防范（维护者指南）

中文 · [English](MAINTAINING.en.md)

本文整理**维护者**在跟进原仓库 [Autumn-27/ARTEX](https://github.com/Autumn-27/ARTEX) 的变更、
维持中文本地化的过程中要走的流程。贡献范围、法律责任与本地化方针见
[CONTRIBUTING.md](CONTRIBUTING.md)，面向使用者的说明见 [README.md](README.md)，
所以本文只聚焦于**这些方针实际怎么执行**。

本地化的核心目标可以概括成一句话：在上游**判断性能**原样保留的前提下，把面向用户的产出统一为
**简体中文**。上游每次更新都容易让这条边界变模糊，因此用下面的流程与检查来防止翻译漂移。

---

## 1. 本地化结构一览

本仓库是在上游 ARTEX 上 **fork**，并在其历史之上叠加中文化提交的结构。上游 `main` 的每个提交
都包含在本仓库的历史里，中文化提交叠加在其上。因此引入上游变更这件事，就是「查看与上游 `main`
的差异，区分该保留的与该翻译的，再落地」。

产物分为三类。

- **原文保持不动的资产**（见第 2 节）。翻译会破坏性能或与上游的对照。
- **代码固定的输出语言强制**。`agent/prompt.go` 的 `langDirective()` 会在各角色 system
  提示词末尾追加「用户可见输出用简体中文书写」的指示。
- **需要中文化的用户可见字符串**。界面文案放在 `web/messages/zh.json`，服务端的用户响应
  文案放在各 Go 文件的具名常量里。

---

## 2. 保留原文的资产（禁止翻译）

以下资产不翻译，保持原文（中文或英文）。上游变更触及这些资产时**不做翻译、原样落地**。

- **agent 内部推理提示词（大脑正文）。** 位于 `agent/promptcatalog.go` 与数据库种子
  `agent_prompts` 的行为指令正文。必须保留按原文（中文）基准测试过的行为，翻译会让判断
  产生漂移。
- **既用于展示、又作为 agent 输入的字符串。** 有些文案（任务中止原因、拦截阻断消息、流量
  证据辅助信息等）既出现在活动时间线上，又回喂给规划者、报告者的输入上下文——同一条记录
  兼作两种用途，因此保留原文。每条这样的取舍都写在引入它的那次提交信息里，需要追溯时用
  `git log -S'<该字符串>'` 查看。
- **上游中文文档与界面文案。** 文档把上游中文原文保留在 `README.zh.md`，便于与上游变更对照；
  界面文案集中放在 `web/messages/zh.json`，组件按消息键取用，不要硬编码。
- **命令、payload、代码、URL、标识符、日志原文。** 它们是分析所需的原始材料，不翻译。
  Go 代码注释的优先级最低，等与上游对照结束后再处理。

---

## 3. 上游追踪基线

上游远端需要按下面这样配置。没有就补上。

```bash
git remote add upstream https://github.com/Autumn-27/ARTEX
git remote -v   # 确认能看到 upstream
```

当前中文化已经跟进到的上游基线提交如下。

- **基线 = `d003372`**（上游 `main`，2026-10-03，PR #189 `fix/sse-same-origin` 合并）。

这个值的含义是「到这个提交为止的上游变更都已并入本仓库」。每次引入新的上游变更后，按第 7 节的
方法更新这个基线。

---

## 4. 引入上游变更的流程

### 4.1 拉取上游并查看差异

```bash
git fetch upstream
git rev-list --count d003372..upstream/main        # 尚未落地的上游提交数
git log --oneline d003372..upstream/main           # 尚未落地的提交列表
```

`git fetch` 只更新上游的远程跟踪分支，不会影响工作树与 `HEAD`。未落地的提交数为 0 就说明与
上游已同步，没有别的事要做。

### 4.2 对变更文件分类

看未落地的提交改了哪些文件，按第 2 节的保留资产与翻译对象分开。

```bash
git log --name-status --oneline d003372..upstream/main
```

分类标准如下。

- `agent/promptcatalog.go`、`agent_prompts` 种子，以及第 2 节里兼作展示与输入的字符串有改动
  → **不做翻译、原样落地**。
- Go 后端逻辑（`db/`、`llmrec/`、`server/` 等）有改动 → 逻辑原样落地，但要确认是否**新增了
  用户响应文案**（`writeErr` 等），并译成中文常量。
- UI（`web/src/**`）有改动、**出现新的界面字符串** → 不要硬编码，在 `web/messages/zh.json` 里
  按同一套键添加。
- **检测规则固定的上游指标**（`enrich/enrich.go` 的探测器 User-Agent、`selfupdate/` 的自更新
  User-Agent、`guard/guard.go` 的审计标记、`db/db.go` 的破坏性命令 deny 列表、
  `cmd/artex/main.go` 的默认监听与记录代理端口）有改动 → 把 `detections/` 的 Sigma、Suricata
  规则与 ATT&CK 图层，以及 `detections/indicators/artex_indicators.csv` 里的值也改成新值。
  这些指标不是翻译对象，而是**检测的依据**，上游改了值而规则不动，规则就会悄悄失效。
  5.4 的指标一致性测试会自动抓住这种错位。

### 4.3 落地

按功能单位合并或挑选落地，然后把 4.2 里挑出的新字符串译成中文。合并过程中
`web/messages/zh.json` 很容易出现缺键，或让上游原文漏进用户可见的位置，所以落地之后必须执行
第 5 节的检查。

> **示例（2026-10-05 时尚未落地的提交）。** `git fetch upstream` 的结果显示上游 `main` 已经推进到
> `b55ceb1`，相对基线 `d003372` 有两个提交（`86729b6` 模型回退审批的 token 计量功能 +
> 合并提交 `b55ceb1`）未落地。这些提交改动了 `db/llm_usage.go`、`llmrec/llmrec.go`、
> `server/intercept.go`、`server/server.go` 这类 Go 逻辑，以及
> `web/src/app/(main)/system/intercept/page.tsx`、`web/src/lib/api.ts`、
> `web/src/lib/mock/handler.ts`、`web/src/lib/types.ts`。
> 因此维护者把 Go 逻辑原样落地，只把 intercept 设置页新增的界面字符串抽成
> `web/messages/zh.json` 的键并中文化即可。（写本文时这两个提交还没有落地，所以基线仍留在
> `d003372`。）

---

## 5. 翻译对称与漂移检查

上游落地或翻译工作之后，确认以下几项。

### 5.1 界面文案目录与韩文残留检查

本仓库的界面固定为简体中文，界面文案只有 `web/messages/zh.json` 一份目录（`web/src/i18n/config.ts`
的 `LOCALES` 只有 `zh`）。改动界面后要确认它是合法的 JSON、键没有重复，并且没有把韩文带进来。
仓库把「不允许韩文残留」固化成了可执行检查 `scripts/check-no-korean.py`：它扫描工作树下的文本文件
（跳过 `.git`、`node_modules` 等依赖目录、构建产物与二进制文件），只要任何一行还有谚文就以非零
退出码失败。上游落地与冲突消解时很容易把历史遗留的韩文重新带进来，所以合并后立刻跑一遍。

```bash
python3 -I scripts/check-no-korean.py          # 全仓库扫描
python3 -I scripts/check-no-korean.py --quiet  # 只输出结论
```

通过时输出 `[+] 未发现韩文残留。` 且退出码为 0；失败时逐行列出文件名、行号与命中内容。
本地与远端用同一条命令，改动后跑一遍即可复现这条门禁。

### 5.2 大脑资产原文保留检查

大脑正文保持中文原文，所以下面这条检查里**汉字行数掉到 0** 反而是大脑被误翻译、遭到污染的
信号。

```bash
python3 -c "import re; han=re.compile(r'[㐀-鿿]'); t=open('agent/promptcatalog.go').read(); print('promptcatalog.go CJK lines =', sum(1 for l in t.splitlines() if han.search(l)))"
```

基准值（2026-10-05）：`promptcatalog.go CJK lines = 70`。这个数大幅减少时，确认大脑正文是否
被翻译了。

### 5.3 构建产物不泄漏非中文文案

界面静态导出后，如果预渲染 HTML 里还能看到韩文，就说明有漏译或残留。构建产物
（`web/out`、`server/webui/dist`）不入库，`check-no-korean.py` 也会跳过它们，所以这一步要在
导出之后单独确认。

```bash
cd web && npm ci && NEXT_EXPORT=1 npm run build   # 生成 out/
# 确认 out/**/*.html 的可见文本里没有谚文
```

### 5.4 检测指标是否仍与上游源码一致

`detections/` 的规则依据的是上游实际输出的字符串（探测器 User-Agent、自更新 User-Agent、审计
标记、破坏性命令 deny 列表）。上游重新同步改掉这些值时，翻译检查会全部通过，而已经发布的规则
却会悄悄停止匹配。下面的测试会双向确认每个指标是否仍同时存在于上游源码与规则中，所以在重新
同步之后要一并运行。

```bash
detections/tests/indicators/run.sh   # 用 Docker 隔离运行，RESULT: PASS 即一致
```

失败时会输出哪个指标错位、错在哪个方向（上游源码变了，还是规则变了），于是按 4.2 最后一条
分类标准把规则与图层对齐到新值。这条测试也会在仓库 CI
（[`.github/workflows/detections.yml`](.github/workflows/detections.yml)）里，于规则树或上述上游源
文件发生改动的每次推送与 PR 上自动运行，把重新同步的漂移挡在合并门禁上。

新增指标时如果**固定了新的上游源文件**（例如加 `cmd/artex/main.go` 的端口指标时），必须把这个
文件也加进上面工作流的 `push`、`pull_request` `paths` 过滤条件里。漏掉的话，只改这个源文件的
PR 就不会触发指标测试，漂移会悄悄通过合并门禁。这个同步本身也由指标测试自动检查（第五项检查
"CI triggers this test when any pinned source changes"）：测试读取的所有非 `detections/` 源码
如果没有在两侧 `paths` 块里列全，测试就会失败，因此源码固定与 CI 触发条件不会在错位的状态下
被合并。

### 5.5 升级检测测试工具版本时

检测测试以固定版本运行 `sigma-cli`、SigmaHQ 校验器插件（`pySigma-validators-sigmahq`）与
Suricata 镜像（默认值在各 `run.sh` 里，可用环境变量覆盖）。升级这些版本可能带来**工具侧的
漂移**，而不是上游源码的变化。尤其 SigmaHQ 校验器每次升级都会增加新的惯例检查，
`detections/tests/sigma_lint/run.sh` 可能因此把新问题暴露成红色。这时要么让规则符合新惯例，
要么在该惯例不适用于单规则集时写明理由，加进
[`detections/tests/sigma_lint/validators.yml`](detections/tests/sigma_lint/validators.yml) 的排除
列表。后端插件调整支持范围时，`sigma_backends` 测试会给出同样的信号。

---

## 6. 用构建与测试做收尾验证

落地与翻译之后，按 [CONTRIBUTING.md 的开发环境](CONTRIBUTING.md#开发环境) 的流程验证后端与前端。
本机没有 Go 时可以用 Docker 做同样的事。

```bash
docker run --rm -v "$PWD":/src -w /src \
  -v artexcn-gomod:/go/pkg/mod -v artexcn-gocache:/root/.cache/go-build \
  golang:1.26 sh -c 'go build ./... && go vet ./... && go test ./... -count=1'
```

翻译用户可见文案时，要一并留下断言该文案的回归测试（`*_localized_test.go`），这样以后上游变更
再次改动这些文案时，测试能立刻发现并提醒同步调整。翻译验证必须使用能力足够的（前沿级）模型。
低价、小模型的输出可能退回原文，不能只凭它的输出来判断翻译是否生效。

---

## 7. 基线更新记录

上游变更落地并验证完成后，把本文第 3 节的**基线提交值更新为新的上游提交**，并把这个改动放进
同一个提交或紧随其后的提交里。这样下一位维护者就能在本文一处看清「已经跟进到哪里」。

提交信息遵循 [CONTRIBUTING.md 的提交信息规则](CONTRIBUTING.md#提交信息)。例如上游同步提交
可以这样写。

```
chore(upstream): 落地上游 d003372..b55ceb1（intercept token 计量）+ 新界面文案中文化
```

---

## 8. 审查与验证守则（常见陷阱）

下面两条是维护者在检查上游落地、翻译与文档补充时反复踩到的陷阱。两者都属于「检查方法本身不对，
把完好的东西误判成坏的」，所以固化成守则，避免不必要的回滚。

### 8.1 查看仓库 CI 状态时要指定仓库

本仓库是上游 ARTEX 的 fork，所以本地 `git remote` 里同时登记了 `origin`（dami9527/artex-cn）与
`upstream`（Autumn-27/ARTEX）（见第 3 节）。在这种状态下，给 `gh` 命令不指定仓库，`gh` 会
**默认选中上游仓库**，显示的是没有我们工作流的上游运行结果。于是可能看着上游 CI 是绿的而
**误以为我们的 CI 通过了**，也可能把我们的工作流（`ci.yml`、`detections.yml`）误判为
"HTTP 404 … not found"。

所以查看 CI 时始终显式指定仓库。

```bash
gh run list -R dami9527/artex-cn --workflow ci.yml --limit 5
gh run list -R dami9527/artex-cn --workflow detections.yml --limit 5
```

设置一次之后，可以省掉 `-R` 并让 `gh` 默认指向本仓库。不过这个设置属于**本地 gh 配置**，不会
提交进仓库，换新机器或新检出后需要重新指定。

```bash
gh repo set-default dami9527/artex-cn
gh repo set-default --view   # 确认显示的是 dami9527/artex-cn
```

### 8.2 文档的外部链接按浏览器方式用 GET 检查

防御指南（[`docs/defense-zh.md`](docs/defense-zh.md) 与 [`docs/defense-en.md`](docs/defense-en.md)）
的第 7 节列出了境内官方渠道（国家互联网应急中心 CNCERT/CC、国家互联网信息办公室、12377 举报
中心等）的链接。确认这些链接是否存活时，只用 `curl -I`（HEAD 请求）或默认 User-Agent，会
**把完好的链接误判成坏链**。部分政务与安全机构站点会因为下面三类原因拒绝这种简单确认。

- **拒绝 HEAD 请求。** 只接受 GET，`curl -I`（HEAD）会返回 4xx。
- **拦截默认 `curl` User-Agent。** 用默认 UA 发 GET 也会被拒，换成浏览器 UA 才返回 200。
- **跳转到别的地址。** 有的站点会先从 `www` 域名跳到带路径前缀的地址，不跟随重定向就会错过
  最终状态。

因此链接确认要**用浏览器 User-Agent、发 GET、跟随重定向**。

```bash
UA='Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0 Safari/537.36'
for u in https://www.cert.org.cn https://www.cac.gov.cn https://www.12377.cn; do
  curl -sS -L -A "$UA" -o /dev/null -w "$u -> %{http_code} %{url_effective}\n" "$u"
done
```

最终状态码是 200 就说明链接有效。状态码出现 400、403 时，先怀疑**是确认方式被服务器的访问
策略挡住了**，而不是链接坏了，再逐一排除 HEAD、默认 UA、未跟随重定向这些因素后重新确认。
外部链接的存活状态会随时间变化，结论以检查当时的实际情况为准。

这条手工流程由 `scripts/check-external-links.py` 原样自动化。它会收集所有被跟踪 `.md` 里代码
围栏、行内代码之外的外部链接（排除保留域名与占位主机），按上面的方式用浏览器 UA、GET、跟随
重定向确认状态，并对网络错误、5xx、429 重试，以区分瞬时抖动与真正的故障。结果分为四类：
OK（2xx·3xx）· RESTRICTED（401·403·405·429，主机活着、只是确认方式被挡）· ALLOWED
（写在 `scripts/external-links-allowlist.txt` 里、我们修不了的上游继承死链）·
DOWN（404·410·5xx·连接错误，很可能已经坏掉）。

- 不联网、只预览检查对象：`python3 -I scripts/check-external-links.py --list`
- 发布与周期检查（出现新坏链时以非零码退出）：`python3 -I scripts/check-external-links.py --strict`

外部链接的存活是 flaky 的，所以**不放进合并门禁**。取而代之的是非阻塞工作流
[`external-links`](.github/workflows/external-links.yml) 每周一与手动触发运行 `--strict`，
一旦出现不在 allowlist 里的 DOWN 就报红。上游原文保留文件继承下来的死链（例如
`CHANGELOG.zh.md` 里致谢的、已经消失的贡献者账号）我们修不了，所以连同理由写进 allowlist，
从 strict 检查里排除。

---

## 9. 发布流水线

推送版本标签（`v*`）时，[`.github/workflows/release.yml`](.github/workflows/release.yml) 会为五个
平台构建二进制，并在满足条件时构建多架构 Docker 镜像。本 fork 还没有打过发布标签，这个工作流
一次都没运行过，所以本节整理流水线的前提与产出，以及这些前提与当前仓库结构是否相符。推送标签
会在公开仓库上生成 GitHub Release，因此要等发布决定做出之后再打标签。

### 9.1 如何打发布

推送以 `v` 开头的标签即可触发工作流。

```bash
git tag v0.3.15
git push origin v0.3.15
```

### 9.2 流水线做什么

工作流分为五个作业。

- **frontend。** 把前端静态导出一次（`web/out`），并把产物作为 `web-dist` 工件上传。下面的
  binaries 作业对每个目标重新取用这份产物。
- **binaries。** 交叉编译五个目标（linux amd64、arm64，darwin amd64、arm64，windows amd64），
  并按目标打成 zip。linux amd64 二进制会跑一次 `artex -h` 冒烟测试，确认二进制真的能执行。
- **release。** 收集全部 zip，生成 `SHA256SUMS` 校验和，创建 GitHub Release 并附上 zip 与校验和。
- **docker-gate。** 检查 `DOCKERHUB_USERNAME`、`DOCKERHUB_TOKEN` 密钥是否配置，并把结果作为下
  一个作业的执行条件传下去。
- **docker。** 只在上述密钥存在时运行，取 binaries 交叉编译出的 linux 二进制，构建多架构镜像并
  推送到 Docker Hub。没有密钥时跳过这个作业，发布 CI 不会报红失败，只产出二进制发布。

### 9.3 构建前提是否与仓库结构相符

流水线建立在下面三个前提之上，我们已在本地直接复现 binaries 作业，确认这些前提与当前仓库结构
全部相符。

- **前端内嵌。** frontend 作业上传的 `web-dist`（即 `web/out` 的内容）由 binaries 作业取到
  `server/webui/dist`，再由 `server/webui_embed.go` 的 `//go:embed all:webui/dist` 把该位置内嵌
  进二进制。所以 binaries 作业不再重新构建前端，而是以 `ARTEX_SKIP_FRONTEND=1` 调用
  [`build.sh`](build.sh)。
- **二进制与打包路径。** `build.sh --target <os>/<arch>` 生成
  `dist/artex-<os>-<arch>/artex` 二进制与 `dist/` 下的 zip 包。zip 里除二进制外还包含启动脚本
  （Linux、macOS 是 `start.sh`，Windows 是 `start.bat`）、`skills/`、`config.example.json` 与
  `README.md`。
- **Docker 镜像的二进制拷贝。** binaries 作业把 linux 二进制单独作为 `bin-linux-<arch>` 工件
  上传，docker 作业把它取到 `dist/<arch>/artex`。[`Dockerfile`](Dockerfile) 的
  `COPY dist/${TARGETARCH}/artex` 会用在多架构构建里由 buildx 按各平台填好的 `TARGETARCH` 命中
  这个路径。[`.dockerignore`](.dockerignore) 没有排除 `dist/`，所以二进制会进入构建上下文。

### 9.4 尚未决定的事项：Docker 镜像命名空间

docker 作业目前把镜像名留在上游的 `autumn27/artex`，而本 fork 要用哪个命名空间发布仍是
待决事项。在决定之前不配置 Docker Hub 密钥，期间的发布
只产出二进制 zip 与校验和（docker 作业跳过）。

### 9.5 不打标签在本地预验证

想在不打公开发布的前提下只确认流水线前提，就在本地复现 binaries 作业。本机没有 Go 时可以用
Docker 做同样的事。

```bash
# 1) 前端静态导出（对应 release.yml 的 frontend 作业）
cd web && npm ci && npm run build:static && cd ..
# 2) 放到 binaries 作业接收工件的位置
rm -rf server/webui/dist && mkdir -p server/webui/dist && cp -a web/out/. server/webui/dist/
# 3) 只按 binaries 作业相同的环境构建一个目标
docker run --rm -v "$PWD":/app -w /app \
  -e ARTEX_SKIP_FRONTEND=1 -e ARTEX_SKIP_NPM_CI=1 \
  -e ARTEX_COMPRESS=0 -e ARTEX_PACKAGE=1 -e ARTEX_PACKAGE_DIR=dist \
  -e ARTEX_BUILD_VERSION=v0.0.0-local \
  golang:1.26 bash -c 'apt-get update && apt-get install -y zip && ./build.sh --target linux/amd64'
# 4) 确认产物：dist/artex-linux-amd64/artex · dist/*.zip · dist/SHA256SUMS
```

`dist/artex-linux-amd64/artex` 是静态链接的 ELF，带 `-h` 时会打印用法后以退出码 0 结束。这正是
binaries 作业的冒烟测试所确认的行为。构建产物（`dist/`、`server/webui/dist/`）不提交进仓库
（已由 `.gitignore` 排除）。
