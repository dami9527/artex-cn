package config

// 启动阶段（建立数据库连接之前）运维人员可见消息的翻译表。这个时点还读不到 settings
// 表，所以不能按运行期设置（ActiveLanguage）取语言，只能按配置的默认语言
// （Language()：环境变量 ARTEX_LANG > 配置文件 > "zh"）来替换。排障所需的标识符
// （ARTEX_PG_DSN、database、dsn、host/user/dbname 等）不翻译，原样保留。
//
// 本仓库的目录一律以简体中文原文为键：中文就是键本身，因此不再重复存 "zh" 条目，
// 值里只放 "en"/"es"。查找失败、或当前语言就是中文时，返回键本身（也就是中文原文）。

var cfgCatalog = map[string]map[string]string{

	"环境变量 ARTEX_PG_DSN": {"en": "environment variable ARTEX_PG_DSN", "es": "variable de entorno ARTEX_PG_DSN"},

	"配置文件 %s (database.dsn)": {"en": "config file %s (database.dsn)", "es": "archivo de configuración %s (database.dsn)"},

	"配置文件 %s (database 字段)": {"en": "config file %s (database fields)", "es": "archivo de configuración %s (campos database)"},

	"未找到数据库配置：环境变量 ARTEX_PG_DSN 未设置，且配置文件 %s 未提供 database（dsn 或 host/user/dbname）。请创建该配置文件或设置环境变量后重试": {"en": "database configuration not found: environment variable ARTEX_PG_DSN is not set and config file %s does not provide database (dsn or host/user/dbname). Create that config file or set the environment variable, then try again", "es": "no se encontró la configuración de la base de datos: la variable de entorno ARTEX_PG_DSN no está definida y el archivo de configuración %s no proporciona database (dsn o host/user/dbname). Cree ese archivo de configuración o defina la variable de entorno e inténtelo de nuevo"},
}

// logCatalog 保存运维人员可见的控制台日志模板（启动期 + 运行期诊断）。
// 键是中文原文模板；格式动词（%s、%d、%v 等）在译文里保持相同的位置顺序，
// 这样 fmt.Printf/log.Printf 的参数仍然对得上。中文即键本身，故值里只放 "en"/"es"。
var logCatalog = map[string]map[string]string{

	"[config] 配置文件: %s": {"en": "[config] config file: %s", "es": "[config] archivo de configuración: %s"},

	"[config] 配置文件: %s (文件不存在 · 仅使用环境变量 ARTEX_PG_DSN)": {"en": "[config] config file: %s (not found · using only environment variable ARTEX_PG_DSN)", "es": "[config] archivo de configuración: %s (no encontrado · usando solo la variable de entorno ARTEX_PG_DSN)"},

	"[config] skill 目录: %s": {"en": "[config] skill directory: %s", "es": "[config] directorio de skills: %s"},

	"已暂停": {"en": "paused", "es": "pausada"},

	"已继续": {"en": "resumed", "es": "reanudada"},

	"[auth] 已将 JWT 密钥从 %s 迁移到 %s（移出可被发现的工作区）": {"en": "[auth] moved JWT key from %s to %s (out of the discoverable workspace)", "es": "[auth] clave JWT movida de %s a %s (fuera del espacio de trabajo detectable)"},

	"[auth] 已将新的 JWT 密钥写入 %s": {"en": "[auth] wrote new JWT key to %s", "es": "[auth] nueva clave JWT escrita en %s"},

	"[auth] %s 少于 %d 个字符，已忽略。请改用设置令牌方式初始化": {"en": "[auth] ignoring %s because it is shorter than %d characters; initialise with the setup token instead", "es": "[auth] se ignora %s porque tiene menos de %d caracteres; inicialice con el token de configuración"},

	"[auth] 生成 %s 哈希失败: %v": {"en": "[auth] failed to hash %s: %v", "es": "[auth] no se pudo generar el hash de %s: %v"},

	"[auth] 保存 %s 失败: %v": {"en": "[auth] failed to store %s: %v", "es": "[auth] no se pudo guardar %s: %v"},

	"[auth] 已通过环境变量 %s 初始化管理员密码（用户名为 ARTEX）": {"en": "[auth] initialised the admin password from environment variable %s (username ARTEX)", "es": "[auth] contraseña de administrador inicializada desde la variable de entorno %s (usuario ARTEX)"},

	"[auth] 管理员密码尚未设置。设置令牌取自环境变量 %s 的值（不输出到日志）": {"en": "[auth] the admin password has not been set yet. The setup token is taken from the value of environment variable %s (not printed to the log)", "es": "[auth] aún no se ha establecido la contraseña de administrador. El token de configuración se toma del valor de la variable de entorno %s (no se muestra en el registro)"},

	"[auth] 生成设置令牌失败: %v": {"en": "[auth] failed to generate the setup token: %v", "es": "[auth] no se pudo generar el token de configuración: %v"},

	"[auth] 管理员密码尚未设置。设置令牌: %s": {"en": "[auth] the admin password has not been set yet. Setup token: %s", "es": "[auth] aún no se ha establecido la contraseña de administrador. Token de configuración: %s"},

	"[auth] 请在首屏(/setup)的「设置令牌」输入框中填入上面的值，或在 POST /api/auth/init 的请求体中作为 setup_token 发送。该令牌不会通过网络传输，只能在此控制台日志中看到": {"en": "[auth] enter the value above in the \"Setup token\" field on the first screen (/setup), or send it as setup_token in the POST /api/auth/init body. This token is never sent over the network and is only visible in this console log", "es": "[auth] introduzca el valor anterior en el campo \"Token de configuración\" de la primera pantalla (/setup) o envíelo como setup_token en el cuerpo de POST /api/auth/init. Este token nunca se transmite por la red y solo puede verse en este registro de consola"},

	"[auth] 警告: 管理员密码尚未设置，而服务器绑定在 %q，其他主机也能访问。没有设置令牌无法完成初始化，但在初始设置完成前，请不要把这个端口暴露到外部网络（默认值为 127.0.0.1:8787）": {"en": "[auth] warning: the admin password has not been set yet and the server is bound to %q, so other hosts can connect. Initialisation still requires the setup token, but do not expose this port to external networks until initial setup is complete (default 127.0.0.1:8787)", "es": "[auth] advertencia: aún no se ha establecido la contraseña de administrador y el servidor está enlazado a %q, por lo que otros hosts pueden conectarse. La inicialización sigue requiriendo el token de configuración, pero no exponga este puerto a redes externas hasta completar la configuración inicial (predeterminado 127.0.0.1:8787)"},

	"[auth] 拒绝初始设置: 设置令牌不匹配 (client=%s)": {"en": "[auth] initial setup rejected: setup token mismatch (client=%s)", "es": "[auth] configuración inicial rechazada: el token de configuración no coincide (client=%s)"},

	"[auth] 已初始化管理员密码 (client=%s)": {"en": "[auth] admin password initialised (client=%s)", "es": "[auth] contraseña de administrador inicializada (client=%s)"},

	"[auth] 拒绝修改密码: 当前密码不匹配 (client=%s)": {"en": "[auth] password change rejected: current password mismatch (client=%s)", "es": "[auth] cambio de contraseña rechazado: la contraseña actual no coincide (client=%s)"},

	"[auth] 已修改管理员密码，所有既有会话都将失效 (client=%s)": {"en": "[auth] admin password changed; all existing sessions are invalidated (client=%s)", "es": "[auth] contraseña de administrador cambiada; todas las sesiones existentes quedan invalidadas (client=%s)"},

	"[auth] 登录失败 (client=%s)": {"en": "[auth] login failed (client=%s)", "es": "[auth] inicio de sesión fallido (client=%s)"},

	"[notify] 后台通知投递循环因 %s 已关闭(仅测试用) — 漏洞 IM 通知不会发送": {"en": "[notify] background notification delivery loop disabled via %s (test only) — vulnerability IM alerts will not be sent", "es": "[notify] el bucle de entrega de notificaciones en segundo plano se desactivó mediante %s (solo pruebas) — no se enviarán alertas IM de vulnerabilidades"},

	"[mcp] %s 发现 %d 个工具并已缓存": {"en": "[mcp] from %s, discovered and cached %d tool(s)", "es": "[mcp] desde %s, se descubrieron y almacenaron en caché %d herramienta(s)"},

	"[mcp] 启动自动发现: 读取列表失败: %v": {"en": "[mcp] auto-discovery at startup: failed to read the list: %v", "es": "[mcp] descubrimiento automático al inicio: no se pudo leer la lista: %v"},

	"[mcp] 启动自动发现 %s 失败: %v": {"en": "[mcp] auto-discovery of %s at startup failed: %v", "es": "[mcp] falló el descubrimiento automático de %s al inicio: %v"},

	"[tools] 已将 orchestration/platform 工具 schema 刷新为代码默认值（一次性）": {"en": "[tools] refreshed orchestration/platform tool schemas to code defaults (one-time)", "es": "[tools] esquemas de herramientas orchestration/platform actualizados a los valores predeterminados del código (una sola vez)"},

	"[reporter] 已预置「%s」agent + finding 触发器": {"en": "[reporter] pre-configured the \"%s\" agent + finding trigger", "es": "[reporter] agente \"%s\" + disparador de finding preconfigurados"},

	"[custom-tool] 自动检测到 python 解释器: %s": {"en": "[custom-tool] auto-detected python interpreter: %s", "es": "[custom-tool] intérprete de python detectado automáticamente: %s"},

	"[pg] 数据库配置来源: %s": {"en": "[pg] database configuration source: %s", "es": "[pg] origen de la configuración de la base de datos: %s"},

	"[mcp] browser MCP 已挂捕获代理 %s (CA %s)": {"en": "[mcp] browser MCP capture proxy attached: %s (CA %s)", "es": "[mcp] proxy de captura de browser MCP conectado: %s (CA %s)"},

	"[mcp] browser MCP 已移除捕获代理配置": {"en": "[mcp] browser MCP capture proxy configuration removed", "es": "[mcp] configuración del proxy de captura de browser MCP eliminada"},
}

// trCfg 把启动期/运维人员可见消息翻译成配置的默认语言（Language()）。格式模板同理：
// 先把模板交给 trCfg，再交给 fmt.Sprintf/fmt.Errorf。
func trCfg(src string) string {
	if m, ok := cfgCatalog[src]; ok {
		if v, ok2 := m[Language()]; ok2 && v != "" {
			return v
		}
	}
	return src
}

// T 把运维人员可见的控制台日志模板翻译成配置的默认语言（Language()），供 cmd 与
// server 的 log.Printf 调用点使用。格式动词按位置保留，所以只包装模板本身：
// log.Printf(config.T(src), args...)。
func T(src string) string {
	if m, ok := logCatalog[src]; ok {
		if v, ok2 := m[Language()]; ok2 && v != "" {
			return v
		}
	}
	return src
}
