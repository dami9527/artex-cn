// 支持的 locale 与默认值。本仓库是 ARTEX 中文版：默认界面语言为简体中文（"zh"），
// 并可通过设置界面切换到英语（"en"）与西班牙语（"es"）。文案集中放在 messages/*.json，
// 组件通过 next-intl 的 useTranslations 取用。
export const LOCALES = ["en", "zh", "es"] as const;
export type Locale = (typeof LOCALES)[number];

// 默认显示语言是简体中文。英语与西班牙语是可在设置中切换的语言。构建时可用
// NEXT_PUBLIC_LOCALE 覆盖；运行期则由用户在设置界面选择的值（localStorage + 同名
// cookie）拥有最终决定权。
export const DEFAULT_LOCALE: Locale = "zh";

// 语言选择器中各 locale 的自称写法。无论当前用哪种语言查看，语言名都应当以该语言
// 自身书写才能被读出来，所以这里不是翻译目标，而是固定写法。
export const LOCALE_LABELS: Record<Locale, string> = {
  en: "English",
  zh: "中文",
  es: "Español",
};

// 保存用户所选显示语言的键。前端读取 localStorage 的值；cookie 里也写入同一份，
// 目前没有读取方（留给将来需要由服务端判定语言的场景）。
export const LOCALE_COOKIE = "artex_locale";
export const LOCALE_STORAGE_KEY = "artex_locale";

// 把任意字符串收窄为受支持的 locale。
export function isLocale(raw: unknown): raw is Locale {
  return typeof raw === "string" && (LOCALES as readonly string[]).includes(raw);
}

// 在构建期决定生效的 locale。必须与静态导出（next.config 的 output: "export"）兼容，
// 所以不读 cookies()/headers() 这类动态 API，只看环境变量。NEXT_PUBLIC_LOCALE 为空
// 或不在支持列表内时回落到默认值。该值只用于服务端预渲染的初始 HTML，真正给用户看的
// 语言由客户端用 localStorage 覆盖。
export function resolveLocale(): Locale {
  const raw = process.env.NEXT_PUBLIC_LOCALE;
  return isLocale(raw) ? raw : DEFAULT_LOCALE;
}
