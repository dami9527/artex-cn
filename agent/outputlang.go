package agent

// 面向用户的输出语言（展示语言）选择。artex-cn 支持英语、中文、西班牙语三种语言，
// 强制 agent 以其中一种书写给用户看的自然语言。agent 的「大脑」（经过基准测试的
// 中文推理正文 段[A]）与语言无关，原样保留。这里定的只是用户读到的那层表层的语言。

// OutputLanguage 是服务器注入的钩子，返回由配置（config 的 ARTEX_LANG、运行时
// settings KV）选定的输出语言代码（"en"/"zh"/"es"）。为 nil 或返回空串时回落到
// 下面的默认值。注入方式与 PromptOverride 相同。
var OutputLanguage func() string

// 服务器未注入钩子时（例如只测 agent 包）使用的默认输出语言。默认值是中文
// （本仓库是中文版，配置层的默认值同样是 zh）。英语、西班牙语是可通过设置切换的
// 附加语言。
const defaultOutputLang = "zh"

// SupportedOutputLangs 是可切换的输出语言代码列表。
var SupportedOutputLangs = []string{"en", "zh", "es"}

// normalizeLang 把支持列表之外的代码收敛到默认值。
func normalizeLang(code string) string {
	for _, c := range SupportedOutputLangs {
		if c == code {
			return code
		}
	}
	return defaultOutputLang
}

// resolveOutputLang 返回当前应采用的输出语言代码。优先看服务器注入的钩子，
// 没有则用默认值。
func resolveOutputLang() string {
	if OutputLanguage != nil {
		if code := OutputLanguage(); code != "" {
			return normalizeLang(code)
		}
	}
	return defaultOutputLang
}

// langPhrase 是在（用中文写成的）规约正文里指认「用这种语言书写」时的写法。
// 括号里附上该语言的自称，避免模型分不清指的是哪种语言。
var langPhrase = map[string]string{
	"zh": "中文（简体中文）",
	"en": "英语（English）",
	"es": "西班牙语（Español）",
}

// langBare 是在规约正文里反复指认时用的简短语言名。
var langBare = map[string]string{
	"zh": "中文",
	"en": "英语",
	"es": "西班牙语",
}
