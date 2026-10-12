# 界面与输出语言（本地化架构）

本文说明 artex-cn 的本地化架构：界面文案、后端消息与 Agent 生成的自然语言分别由谁决定、
新增一条文案要改哪些地方，以及哪些门禁会拦住回归。

**一句话结论**：默认简体中文（`zh`），可切换英语（`en`）与西班牙语（`es`）；
**不存在韩语（`ko`）**，任何地方混入谚文都会被 `scripts/check-no-korean.py` 拦下。

---

## 1. 语言集合

| 位置 | 定义 | 取值 |
| --- | --- | --- |
| 前端 | `web/src/i18n/config.ts` | `LOCALES = ["en", "zh", "es"]`，`DEFAULT_LOCALE = "zh"` |
| 后端 | `config/config.go` | `SupportedLanguages = ["en", "zh", "es"]`，`DefaultLanguage = "zh"` |
| Agent 输出 | `agent/outputlang.go` | `SupportedOutputLangs = ["en", "zh", "es"]`，`defaultOutputLang = "zh"` |

三者必须保持一致。语言代码只有这三个；`config.NormalizeLanguage` 与前端 `isLocale` 会把
列表之外的取值收敛到默认值。

---

## 2. 前端（Next.js + next-intl）

### 2.1 文件

| 文件 | 作用 |
| --- | --- |
| `web/messages/zh.json` · `en.json` · `es.json` | 三份文案，键路径必须完全一致（当前各 3013 个叶子键） |
| `web/src/i18n/config.ts` | 语言集合、默认值、`LOCALE_LABELS`、存储键、`isLocale`、`resolveLocale` |
| `web/src/i18n/locale-store.ts` | 运行期切换：`useActiveLocale()` / `setActiveLocale()`，写 `localStorage` + 同名 cookie |
| `web/src/i18n/intl-runtime-provider.tsx` | 按当前 locale 注入 `messages`，同步 `<html lang>` |
| `web/src/i18n/request.ts` | 服务端请求配置，决定预渲染用的默认语言 |

### 2.2 构建期与运行期

- **构建期**：`resolveLocale()` 只读 `NEXT_PUBLIC_LOCALE`，不读 `cookies()`/`headers()`，
  因此与静态导出（`output: "export"`）兼容。它只决定预渲染 HTML 的初始语言。
- **运行期**：用户在「系统 → 设置 → 显示语言」里选择的值存在 `localStorage`（键
  `artex_locale`）并同步写入同名 cookie，由 `IntlRuntimeProvider` 立即切换，无需刷新或重新构建。
  三份文案都被静态打进包里（`MESSAGES`），所以切换是同步的。
- **404 页面**：`web/src/app/global-not-found.tsx` 不继承根布局，必须自己把 `messages`
  传给 `NextIntlClientProvider`。它只挑 `notFound` 命名空间以减小体积，但**三种语言都必须有**，
  否则 `MESSAGES[locale]` 会是 `undefined`。

### 2.3 新增一条界面文案

1. 往 `web/messages/zh.json` 加键，写中文；
2. 往 `en.json`、`es.json` 加**同路径**的键（漏了会在切换语言时显示原始键名）；
3. 组件里用 `useTranslations("<命名空间>")` 取用。

插值用 `{name}` 形式，三份文件的占位符集合必须一致。

---

## 3. 后端（Go）

后端的做法是 **gettext 风格：把「中文原文」本身当作翻译键**。这一点是本仓库与上游
（`artex-ko`，以韩文原文为键）最关键的区别，也是最容易踩坑的地方。

### 3.1 两个目录

| 目录 | 文件 | 查找函数 | 用途 |
| --- | --- | --- | --- |
| `errCatalog` | `server/errmsg_catalog.go` | `trMsg(src)` | HTTP 响应体里的 `error` 字段（`writeErr` / `writeJSON`） |
| `cfgCatalog` / `logCatalog` | `config/msg_i18n.go` | `trCfg(src)` / `T(src)` | 启动期运维消息；`logT(src)` 是 `config.T` 在 server 包内的别名 |

条目的值是 `{"en": …, "es": …}`。**中文就是键本身，值里不存 `"zh"`**：查找失败或当前语言是
中文时，`trMsg`/`T` 直接原样返回传入的字符串，这正是中文界面想要的结果。

### 3.2 优先级

```
运行期（settings 表的 output_language，用户在界面上选的）
    > config.Language() = ARTEX_LANG 环境变量 > 配置文件 language 字段 > "zh"
```

`config.ActiveLanguage()` 读运行期值（经服务器注入的 `config.CurrentLanguage` 钩子），
回落 `"zh"`；`config.Language()` 读引导默认值。服务器启动时把同一个闭包同时接到
`agent.OutputLanguage`、`config.CurrentLanguage`、`notify.Lang` 上，因此 Agent 生成的文字、
后端固定文案、IM 推送里的级别/状态标签共用同一个语言。用户在界面上改语言后，
`putSettings` 会刷新缓存，**下一轮 Agent 对话即生效，无需重启**。

### 3.3 新增一条后端消息（重要）

**目录键必须与源码里的中文字符串逐字一致。** 措辞差一个字（多一个空格、全角/半角冒号不同）
都会让查找静默落空——中文界面看不出问题，但 en/es 用户会看到中文原文。步骤：

1. 在源码里写常量或字面量，例如 `errFoo = "无法打开配置文件: "`；
2. 在对应目录里加**同一个字符串**作键，给 `en`/`es` 译文：
   `"无法打开配置文件: ": {"en": "Failed to open the config file: ", "es": "No se pudo abrir el archivo de configuración: "}`；
3. 格式动词（`%s`、`%d`、`%w`）必须与原文**同序同数量**，且只包裹模板本身：
   `log.Printf(logT("已处理 %d 项"), n)`；
4. 若把常量经标识符传入（`trMsg(errFoo)`），目录键仍是该常量的取值。

改名文案时**必须同步改目录键**，否则门禁会失败（见第 5 节）。

### 3.4 不需要翻译的内容

命令、payload、代码、文件路径、URL、参数名、日志/请求/响应原文、纯 ASCII 技术串
（`token`、`pause|resume` 等）一律保持原样。目录里也只为「含汉字的面向用户文案」建条目。

---

## 4. Agent 生成文字的语言

`agent/outputlang.go` 提供 `OutputLanguage` 钩子（由服务器注入）。`agent/prompt.go` 的
`langDirective(code)` 会按当前语言生成一段**用中文书写**的「输出语言规约」，追加在每一个
面向用户的角色的 system prompt 末尾（正文 + 产物/流量尾段之后），因此即使有人改了 DB 里的
提示词正文也丢不掉这条约束。规约强制：

- `record_fact` 的 summary/detail、`report_finding` 的标题/描述/结论/修复建议、
  规划者的态势总结、最后的一句话总结、以及对用户的聊天回复，全部用展示语言书写；
- 命令、payload、代码、路径、URL 与日志原文逐字保留，不翻译（保证证据可复现）。

它**不翻译 Agent 的「大脑」**：经过基准测试的中文推理正文（段 [A]）原样保留，只约束用户读到
的表层。`agent/wrapup.go` 里的收尾提示词不会自动继承 system prompt，因此单独追加同一段规约。

---

## 5. 门禁

| 门禁 | 位置 | 断言 |
| --- | --- | --- |
| 韩文残留 | `scripts/check-no-korean.py`（CI：`web.yml`，`working-directory: .`） | 全仓库受控文本文件里不得出现谚文（U+AC00–D7A3、U+1100–11FF、U+3130–318F 等）；跳过 `.git`/`node_modules`/`out`/`dist`/二进制 |
| 后端文案可译 | `server/logt_catalog_test.go` · `TestServerErrorMessagesHaveTranslations` | server 包里所有 `err*` 字符串常量与 `writeErr` 第三个实参子树里的中文字面量，都必须在 `errCatalog` 里有键（豁免需在 `errConstExempt` 里逐条写明理由） |
| 目录无僵尸键 | 同文件 · `TestErrCatalogKeysAreLive` | 目录每个键都必须在源码里真实出现；键/值不得含谚文；en/es 非空；值里不得重复存 `zh` |
| 日志模板可译 | 同文件 · `TestLogTTemplatesHaveTranslations` | `logT(...)` 的中文实参在 `logCatalog` 里有译文；中文时原样返回 |
| 格式动词一致 | `server/logt_catalog_test.go` + `config/msg_i18n_test.go` | 每条译文的格式动词与原文同序同数量 |
| 文案目录结构 | `config/msg_i18n_test.go` · `TestCatalogKeysAreChineseAndTranslated` | 键是中文且不含谚文；每条都有 en/es |
| 前端 | CI `web.yml`：`npm run build:static`（含类型检查）→ 韩文门禁 → `npm run check`（biome，0 错误） | 三份 `messages/*.json` 的键路径必须一致，否则切换语言时会出现原始键名 |

本地按下面的顺序自查，与 CI 一致：

```bash
python3 -I scripts/check-no-korean.py      # 韩文（应输出「未发现韩文残留」）
python3 -I scripts/check-doc-links.py      # 文档内部链接与锚点
go build ./... && go vet ./... && go test ./...
cd web && npm run build:static && npm run check
```

---

## 6. 已知例外：第三方依赖产物里的韩文

构建产物 `web/out/_next/static/chunks/*.js` 里可能出现少量谚文，来源是第三方可视化依赖
（`@antv/g6`、`@antv/g`、`html2canvas`）内置的 **CJK 计数器/数字格式数据**
（如 CSS `korean-hangul-formal` 计数器样式里的韩文「负数」一词）。这不是本仓库的代码、注释、
提示词或入库数据，也无法在不 patch 依赖的前提下移除，重新安装依赖后会再次出现。

仓库自己的约定与门禁均排除依赖与构建产物目录（`node_modules`、`out`、`dist`），
因此这不影响 `scripts/check-no-korean.py` 通过。若日后替换掉这些依赖，该现象会自然消失。

---

## 7. 常见坑

1. **改了文案忘了改目录键** → 中文界面正常，en/es 回落成中文。跑 `go test ./server/` 会被门禁拦下。
2. **目录键全角/半角标点不一致** → 同样是静默落空。复制粘贴源码里的字符串，不要手敲。
3. **只为 `zh.json` 加键** → 切到 en/es 时界面显示 `some.namespace.key` 这样的原始键名。
4. **`logT` 包错实参** → 要翻译哪一段就包哪一段。例如
   `log.Printf("[task] #%s %s", id, logT(label))`（译标签），而不是 `logT("[task] #%s %s")`。
5. **新增面向用户角色时忘了语言尾巴** → 该角色的输出不会被约束到展示语言。新角色的
   system prompt 组装处要像 `workerSystem`/`plannerSystem` 一样追加 `langDirective(...)`。
