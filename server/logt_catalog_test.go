package server

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/Autumn-27/artex/config"
)

// isHan 报告 r 是否是汉字。中文目录的键就是中文原文，所以可翻译的日志模板必然含汉字，
// 扫描也以此识别「需要译文的模板」。
func isHan(r rune) bool { return unicode.Is(unicode.Han, r) }

// TestLogTTemplatesHaveTranslations 钉住本包每个 logT("…") 模板都在 config 的日志目录
// 里有 en/es 条目。键是中文原文、中文即键本身（不另存 zh），所以查找落空时 config.T 会
// 回落成中文原文，把中文控制台日志发给非中文运维；一个字符的键漂移（空格、转义引号）
// 就会造成这种静默漏译，因此必须有测试盯住。
func TestLogTTemplatesHaveTranslations(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var keys []string
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatalf("解析 %s: %v", f, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			if id, ok := call.Fun.(*ast.Ident); !ok || id.Name != "logT" {
				return true
			}
			// 纯字面量就是模板本身；计算出来的实参（例如 map 查表）按其中包含的
			// 每个汉字字面量检查。
			ast.Inspect(call.Args[0], func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				s, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("%s: %v", fset.Position(lit.Pos()), err)
				}
				if strings.IndexFunc(s, isHan) >= 0 {
					keys = append(keys, s)
				}
				return true
			})
			return true
		})
	}
	if len(keys) == 0 {
		t.Fatal("没有找到任何 logT 模板；扫描逻辑已失效")
	}
	for _, k := range keys {
		// zh：键就是中文原文，目录里不重复存 zh，必须原样返回。
		t.Setenv("ARTEX_LANG", "zh")
		if got := config.T(k); got != k {
			t.Errorf("zh 下必须原样返回中文模板: %q -> %q", k, got)
		}
		for _, lang := range []string{"en", "es"} {
			t.Setenv("ARTEX_LANG", lang)
			if config.T(k) == k {
				t.Errorf("logT 模板缺少 %s 译文: %q", lang, k)
			}
		}
	}
}

// TestErrCatalogFormatVerbsMatch 钉住译文与中文模板的格式动词顺序一致；trMsg 先替换
// 模板，再由 fmt.Sprintf/Errorf 填充，顺序错位就会把响应内容弄乱。
func TestErrCatalogFormatVerbsMatch(t *testing.T) {
	verbs := regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z%]`)
	for src, tr := range errCatalog {
		want := strings.Join(verbs.FindAllString(src, -1), " ")
		for lang, v := range tr {
			if got := strings.Join(verbs.FindAllString(v, -1), " "); got != want {
				t.Errorf("errCatalog[%q][%s] 动词 = %q，期望 %q", src, lang, got, want)
			}
		}
	}
}

// hasHangul 报告 s 里是否混入了谚文（U+AC00–U+D7A3 等）。本仓库要求零韩文，
// 目录的键与译文都不允许出现。
func hasHangul(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool { return unicode.Is(unicode.Hangul, r) }) >= 0
}

// errSite 记录一处面向用户的错误文案来自哪里，失败信息里给出「变量名 + 文件:行」，
// 便于直接定位到源码。
type errSite struct {
	name string // 常量/变量名；writeErr 内联字面量记作 "(writeErr 内联)"
	pos  string // file:line
}

// errConstExempt 逐条列出**确实不需要多语言**的错误常量（键为变量名，值为理由）。
// 门禁默认要求每一条含汉字的 err*/authErr* 字符串常量都在 errCatalog 里有键；
// 确实不该翻译的必须在这里显式豁免并写明原因——不允许用宽泛规则整体放过，
// 否则门禁就形同虚设。
var errConstExempt = map[string]string{
	// 目前没有豁免项：server 包里所有含汉字的 err*/authErr* 文案都会经 writeErr
	// 或 SSE/日志展示给用户，都应可译。如日后确有一条仅用于内部包装且永不出现在
	// 用户可见文案里，在此登记并写明依据。
}

// collectErrSources 扫描 server 包里所有面向用户的错误文案，返回：
//   - sites：文案原文 → 来源列表（err*/authErr* 字符串常量的值，以及
//     writeErr(w, code, …) 第三个实参子树里出现的字符串字面量，含 fmt.Sprintf 模板）；
//   - literals：除目录文件 errmsg_catalog.go 之外，源码里出现过的**全部**字符串字面量。
//     用于反向检测「僵尸键」：目录里存在、但源码里任何地方都没有的键，永远查不到。
func collectErrSources(t *testing.T) (map[string][]errSite, map[string]bool) {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	sites := map[string][]errSite{}
	literals := map[string]bool{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatalf("解析 %s: %v", f, err)
		}
		// 目录文件本身不能算「源码里出现过」，否则反向检测会自我印证。
		isCatalog := f == "errmsg_catalog.go"
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.ValueSpec:
				for i, name := range node.Names {
					if i >= len(node.Values) || !strings.HasPrefix(name.Name, "err") {
						continue
					}
					lit, ok := node.Values[i].(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					if s, err := strconv.Unquote(lit.Value); err == nil {
						sites[s] = append(sites[s], errSite{name: name.Name, pos: fset.Position(name.Pos()).String()})
					}
				}
			case *ast.CallExpr:
				if id, ok := node.Fun.(*ast.Ident); !ok || id.Name != "writeErr" || len(node.Args) < 3 {
					return true
				}
				ast.Inspect(node.Args[2], func(n ast.Node) bool {
					lit, ok := n.(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						return true
					}
					if s, err := strconv.Unquote(lit.Value); err == nil {
						sites[s] = append(sites[s], errSite{name: "(writeErr 内联)", pos: fset.Position(lit.Pos()).String()})
					}
					return true
				})
			}
			return true
		})
		if isCatalog {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			if s, err := strconv.Unquote(lit.Value); err == nil {
				literals[s] = true
			}
			return true
		})
	}
	if len(sites) == 0 {
		t.Fatal("没有找到任何错误文案来源；扫描逻辑已失效")
	}
	return sites, literals
}

// allExempt 报告某个文案的所有来源是否都已被 errConstExempt 显式豁免。
func allExempt(locs []errSite) bool {
	for _, l := range locs {
		if _, ok := errConstExempt[l.name]; !ok {
			return false
		}
	}
	return true
}

// TestServerErrorMessagesHaveTranslations 是全量门禁：server 包里所有面向用户的错误
// 文案——err*/authErr* 字符串常量，以及 writeErr(w, code, "…") 内联的中文字面量
// （含 fmt.Sprintf 模板）——都必须能在 errCatalog 里查到键。键是中文原文、中文即键
// 本身（不存 zh），所以查找落空时 en/es 会静默回落成中文原文，这正是本测试要防的
// 回归（原先只 gate 了 authErr*，覆盖不到其余 200 多处）。
// 纯 ASCII 的标识符/技术串（如 "asset_id"、"%w"）没有自然语言可译，不纳入要求；
// 确实不该翻译的常量必须在 errConstExempt 里逐条豁免。
func TestServerErrorMessagesHaveTranslations(t *testing.T) {
	sites, _ := collectErrSources(t)
	var missing []string
	for src, locs := range sites {
		if strings.IndexFunc(src, isHan) < 0 {
			continue // 纯 ASCII 技术串：没有可翻译的自然语言
		}
		if allExempt(locs) {
			continue
		}
		if _, ok := errCatalog[src]; !ok {
			where := make([]string, 0, len(locs))
			for _, l := range locs {
				where = append(where, l.name+" @"+l.pos)
			}
			sort.Strings(where)
			missing = append(missing, fmt.Sprintf("%q\n        来源: %s", src, strings.Join(where, "; ")))
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("以下 %d 条错误文案在 errCatalog 里没有键，en/es 会静默回落成中文：\n    %s",
			len(missing), strings.Join(missing, "\n    "))
	}
}

// TestErrCatalogKeysAreLive 是反向门禁：目录里的每个键都必须（1）不含谚文，
// （2）真的出现在 server 源码中某处（否则就是一个永远查不到的僵尸键），
// （3）带有非空且不含谚文的 en/es 译文。
func TestErrCatalogKeysAreLive(t *testing.T) {
	_, literals := collectErrSources(t)
	var hangulKeys, zombies, badValues []string
	for key, tr := range errCatalog {
		if hasHangul(key) {
			hangulKeys = append(hangulKeys, key)
		}
		if !literals[key] {
			zombies = append(zombies, key)
		}
		for _, lang := range []string{"en", "es"} {
			v, ok := tr[lang]
			if !ok || strings.TrimSpace(v) == "" {
				badValues = append(badValues, fmt.Sprintf("%q 缺少 %s 译文", key, lang))
				continue
			}
			if hasHangul(v) {
				badValues = append(badValues, fmt.Sprintf("%q 的 %s 译文含谚文: %q", key, lang, v))
			}
		}
		if _, ok := tr[config.DefaultLanguage]; ok {
			// 中文即键本身，不该在值里重复存一份。
			badValues = append(badValues, fmt.Sprintf("%q 的值里重复存了 %q 条目（中文即键本身）", key, config.DefaultLanguage))
		}
	}
	sort.Strings(hangulKeys)
	sort.Strings(zombies)
	sort.Strings(badValues)
	if len(hangulKeys) > 0 {
		t.Errorf("errCatalog 有 %d 个键含谚文（必须改成简体中文原文）：\n    %s",
			len(hangulKeys), strings.Join(hangulKeys, "\n    "))
	}
	if len(zombies) > 0 {
		t.Errorf("errCatalog 有 %d 个僵尸键（server 源码里根本不存在，永远查不到）：\n    %s",
			len(zombies), strings.Join(zombies, "\n    "))
	}
	if len(badValues) > 0 {
		t.Errorf("errCatalog 有 %d 条译文不合格：\n    %s",
			len(badValues), strings.Join(badValues, "\n    "))
	}
}
