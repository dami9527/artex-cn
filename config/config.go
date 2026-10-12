// Package config loads runtime configuration from a JSON file, with environment
// variables taking precedence. Currently it carries the PostgreSQL connection.
package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Database is the PostgreSQL connection config. Either set DSN directly, or set
// the component fields and a DSN is assembled from them.
type Database struct {
	DSN      string `json:"dsn"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

// Config is the on-disk config file shape.
type Config struct {
	Database Database `json:"database"`
	SkillDir string   `json:"skill_dir"`
	// Language 是面向用户输出语言的引导默认值（"en"/"zh"/"es"），为空时取
	// DefaultLanguage。运行期以 settings 表中由用户在界面上选定的值为准，覆盖此处。
	Language string `json:"language"`
}

// SupportedLanguages 列出 artex-cn 可切换的面向用户输出语言。agent 输出、界面文案
// 与后端消息都以这些代码为键。
var SupportedLanguages = []string{"en", "zh", "es"}

// DefaultLanguage 是未做任何配置时的开箱默认语言。artex-cn 默认中文；英语与西班牙语
// 可通过语言设置（以及 ARTEX_LANG）在运行期切换。
const DefaultLanguage = "zh"

// NormalizeLanguage lower-cases and validates a language code, returning "" when
// the code is empty or unsupported (so callers can fall through to the next source).
func NormalizeLanguage(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	for _, c := range SupportedLanguages {
		if c == code {
			return c
		}
	}
	return ""
}

// CurrentLanguage 由服务器注入，返回运行期选定（settings 表）并叠加在引导默认值
// 之上的当前面向用户输出语言。后端展示用文案（枚举标签等）一律经 ActiveLanguage
// 读取，它就是唯一来源。为 nil（单元测试／独立运行）时 ActiveLanguage 回落到 "zh"，
// 这样本地化黄金测试不必每条都显式钉住语言。
var CurrentLanguage func() string

// ActiveLanguage 解析后端展示文案所用的语言。优先级：服务器注入的运行时钩子
// （settings 表）高于未注入时的 "zh" 回落值。生产环境总会注入 CurrentLanguage，
// 因此全新部署会经该钩子解析到配置的默认值（DefaultLanguage = "zh"）；"zh" 回落值
// 只在钩子缺失（单元测试／独立运行）时生效。
func ActiveLanguage() string {
	if CurrentLanguage != nil {
		if v := NormalizeLanguage(CurrentLanguage()); v != "" {
			return v
		}
	}
	return "zh"
}

// Language resolves the default user-facing output language with precedence:
//
//	env ARTEX_LANG  >  config file (language)  >  DefaultLanguage
//
// This is the bootstrap default only. The runtime value a user picks in the UI is
// stored in the settings table and takes precedence over this — the server layers
// the two and wires the result into the agent and message catalogs.
func Language() string {
	if v := NormalizeLanguage(os.Getenv("ARTEX_LANG")); v != "" {
		return v
	}
	if v := NormalizeLanguage(Load().Language); v != "" {
		return v
	}
	return DefaultLanguage
}

// BaseDir is the directory that anchors all runtime artifacts (config.json and
// the data/ store). It is the directory the running binary lives in, so a
// distributed executable keeps its files next to itself on any OS (Windows,
// Linux, …) regardless of the working directory it is launched from.
//
// When launched via `go run`, the binary is throwaway: it sits either in a temp
// build dir (cache miss → fresh link) OR straight inside the Go build cache
// (cache hit → run from $GOCACHE/.../...-d). We detect both and fall back to the
// current working directory so dev artifacts (data/, transcripts) and config
// resolve against the project dir, not the throwaway binary's location.
func BaseDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	dir := filepath.Dir(exe)
	if isGoRunDir(dir) {
		return "." // throwaway `go run` binary → use CWD
	}
	return dir
}

// isGoRunDir reports whether dir is where `go run` parked its executable: under
// the system temp dir (cache miss), or anywhere inside a Go build cache
// (…/go-build/…, the cache-hit case — NOT under os.TempDir(), which is why the
// old temp-only check failed intermittently). In both cases the binary is
// throwaway, so config/data must resolve against the CWD.
func isGoRunDir(dir string) bool {
	if tmp := os.TempDir(); tmp != "" {
		if rel, err := filepath.Rel(tmp, dir); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	for _, seg := range strings.Split(filepath.ToSlash(dir), "/") {
		if seg == "go-build" {
			return true
		}
	}
	return false
}

// Path returns the config file path. Resolution order:
//  1. env ARTEX_CONFIG (explicit override)
//  2. ./config.json in the current working directory (running from the project
//     dir — robust no matter where `go run` placed the temp/cached binary)
//  3. config.json next to the executable (a distributed binary keeps it beside)
//
// The first existing file wins. If none exist, the CWD path is returned so the
// "not found" message points at the project dir the user most likely expected.
func Path() string {
	if v := strings.TrimSpace(os.Getenv("ARTEX_CONFIG")); v != "" {
		return v
	}
	var candidates []string
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "config.json"))
	}
	candidates = append(candidates, filepath.Join(BaseDir(), "config.json"))
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return candidates[0]
}

// Load reads and parses the config file. A missing/unreadable file yields a zero
// Config (so callers fall back to defaults) rather than an error.
func Load() Config {
	var c Config
	b, err := os.ReadFile(Path())
	if err != nil {
		return c
	}
	_ = json.Unmarshal(b, &c)
	return c
}

// Locale 返回 UI 显示语言，等同 ActiveLanguage()：运行期由用户在设置里选的
// 语言优先，其次是 ARTEX_LOCALE / 配置文件的 language 字段，最后回落到默认的
// "zh"。
//
// 为什么需要单独一个入口：web UI 的 locale 由客户端决定（web/src/i18n/config.ts
// 的 LOCALE_STORAGE_KEY），而服务器**写入 DB 的显示名称**（reporter、retester
// 这类内置 agent 名）同样显示在界面上，需要跟随同一个语言。部署脚本把构建时的
// locale 经 ARTEX_LOCALE 传进来，避免两处取值不一致。
//
// 种子名称只在语言变化时对齐一次，并且只覆盖仍是历代默认值的那些（用户在 UI 中
// 改过的名称不动）。
func Locale() string {
	return ActiveLanguage()
}

// SkillDir returns the skill root directory with precedence:
//
//	env ARTEX_SKILL_DIR  >  config file (skill_dir)  >  BaseDir()/skills
//
// The directory is created if it does not exist.
func SkillDir() string {
	var d string
	if v := strings.TrimSpace(os.Getenv("ARTEX_SKILL_DIR")); v != "" {
		d = v
	} else if v := strings.TrimSpace(Load().SkillDir); v != "" {
		d = v
	} else {
		d = filepath.Join(BaseDir(), "skills")
	}
	_ = os.MkdirAll(d, 0o755)
	return d
}

// PostgresDSN resolves the connection string with precedence:
//
//	env ARTEX_PG_DSN  >  config file (database.dsn, or assembled from fields)
//
// There is NO built-in fallback: when neither source supplies a database config,
// it returns an error naming the config path it inspected, so startup fails loudly
// instead of silently connecting to a wrong default. source describes where the
// DSN came from (for startup logging).
func PostgresDSN() (dsn, source string, err error) {
	if v := strings.TrimSpace(os.Getenv("ARTEX_PG_DSN")); v != "" {
		return v, trCfg("环境变量 ARTEX_PG_DSN"), nil
	}
	db := Load().Database
	if d := strings.TrimSpace(db.DSN); d != "" {
		return d, fmt.Sprintf(trCfg("配置文件 %s (database.dsn)"), Path()), nil
	}
	if db.Host != "" || db.DBName != "" || db.User != "" {
		return db.buildDSN(), fmt.Sprintf(trCfg("配置文件 %s (database 字段)"), Path()), nil
	}
	return "", "", fmt.Errorf(trCfg("未找到数据库配置：环境变量 ARTEX_PG_DSN 未设置，且配置文件 %s 未提供 database（dsn 或 host/user/dbname）。请创建该配置文件或设置环境变量后重试"), Path())
}

func (d Database) buildDSN() string {
	host := d.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := d.Port
	if port == 0 {
		port = 5432
	}
	ssl := d.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	u := url.URL{
		Scheme: "postgres",
		Host:   host + ":" + strconv.Itoa(port),
		Path:   "/" + d.DBName,
	}
	if d.User != "" {
		if d.Password != "" {
			u.User = url.UserPassword(d.User, d.Password)
		} else {
			u.User = url.User(d.User)
		}
	}
	u.RawQuery = url.Values{"sslmode": {ssl}}.Encode()
	return u.String()
}

// String is a redacted view of the resolved DSN (password masked) for logging.
func Redact(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	if u.User != nil {
		if _, hasPw := u.User.Password(); hasPw {
			u.User = url.UserPassword(u.User.Username(), "****")
		}
	}
	return fmt.Sprintf("%s", u.String())
}
