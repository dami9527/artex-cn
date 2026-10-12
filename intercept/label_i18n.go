package intercept

import "github.com/Autumn-27/artex/config"

// 拦截判定动作标签(allow/deny/ask)的多语言表。用于判定消息，跟随输出语言
// (config.ActiveLanguage)。未注入或语言不受支持时回落到 zh。enum 键(allow/deny/ask)
// 与来源标记("[模型]")是稳定标识符，不翻译。
var judgeActionLabels = map[string]map[string]string{
	"en": {"allow": "allow", "deny": "deny", "ask": "ask"},
	"zh": {"allow": "允许", "deny": "拦截", "ask": "询问"},
	"es": {"allow": "permitir", "deny": "bloquear", "ask": "preguntar"},
}

func judgeActionLang() string {
	lang := config.ActiveLanguage()
	if _, ok := judgeActionLabels[lang]; ok {
		return lang
	}
	return "zh"
}
