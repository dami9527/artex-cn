// 支持的 locale 与默认值。本仓库是 ARTEX 中文版，界面语言固定为简体中文（"zh"）。
// 文案集中放在 messages/zh.json，组件通过 next-intl 的 useTranslations 取用，
// 这样后续要做多语言时只需新增目录文件，不必再改组件。
export const LOCALES = ["zh"] as const;
export type Locale = (typeof LOCALES)[number];
export const DEFAULT_LOCALE: Locale = "zh";

// 在构建期决定生效的 locale。必须与静态导出（next.config 的 output: "export"）
// 兼容，所以不读 cookies()/headers() 这类动态 API，只看环境变量。
// NEXT_PUBLIC_LOCALE 为空或不在支持列表内时回落到默认值（zh）。
export function resolveLocale(): Locale {
  const raw = process.env.NEXT_PUBLIC_LOCALE;
  return LOCALES.includes(raw as Locale) ? (raw as Locale) : DEFAULT_LOCALE;
}
