package server

import "github.com/Autumn-27/artex/config"

// 把用户可见的后端消息（简体中文原文）翻译成当前生效输出语言的目录。本仓库的
// writeErr 响应文句、以及一部分 writeJSON 的 error 字段，都以中文常量/字面量书写
// （F3b/F34）。这里以那些中文原文为键，值为英语、西班牙语译文；中文就是键本身，
// 因此不再重复存 "zh"。查找失败或当前语言是中文时，原样返回原文。
//
// 键是从源码里的中文常量/字面量按字节原样抽取生成的（脚本辅助），不是手敲的，
// 因此 writeErr 运行时拿到的值与键不会错位、导致静默漏译。
//
// 格式字符串常量（%d、%s、%w 等）格式化后的结果与键不同，所以调用方要在把模板交给
// fmt.Sprintf/fmt.Errorf 之前，先用 trMsg(模板) 替换模板本身。

// trMsg 把中文原文消息翻译成当前生效的输出语言。目录里没有、或当前语言就是中文时，
// 原样返回原文。config.ActiveLanguage 反映运行期设置（settings 表），因此用户选定的
// 语言会立即生效。
func trMsg(src string) string {
	if m, ok := errCatalog[src]; ok {
		if v, ok2 := m[config.ActiveLanguage()]; ok2 && v != "" {
			return v
		}
	}
	return src
}

// logT 翻译运维人员可见的控制台日志模板（config.T），让 server 包的 log.Printf 调用点
// 不必各自 import config 就能包装模板。
func logT(src string) string { return config.T(src) }

// loadOutputLanguage resolves the user-facing output language from the source of
// truth: the runtime setting (settings table) over config.Language() (ARTEX_LANG >
// config file > zh). Call it only to (re)fill the cache; it does a DB query and a
// config file read.
func (s *Server) loadOutputLanguage() string {
	if s.m != nil && s.m.pg != nil {
		if v, ok, _ := s.m.pg.GetSetting(settingOutputLanguage); ok {
			if nv := config.NormalizeLanguage(v); nv != "" {
				return nv
			}
		}
	}
	return config.Language()
}

// outputLanguage returns the cached output language, resolving it on first use.
func (s *Server) outputLanguage() string {
	if v, ok := s.outputLang.Load().(string); ok && v != "" {
		return v
	}
	v := s.loadOutputLanguage()
	s.outputLang.Store(v)
	return v
}
