import { NextIntlClientProvider } from "next-intl";

import { APP_CONFIG } from "@/config/app-config";
import { type Locale, resolveLocale } from "@/i18n/config";
import { fontVars } from "@/lib/fonts/registry";
import { PREFERENCE_DEFAULTS } from "@/lib/preferences/preferences-config";
import { ThemeBootScript } from "@/scripts/theme-boot";

import zhMessages from "../../messages/zh.json";
import { GlobalNotFoundBody } from "./_components/global-not-found-body";

import "./globals.css";

// 本页面不继承根布局（它直接返回完整的 HTML 文档），所以必须把 **messages 直接
// 传给** NextIntlClientProvider。不传的话，静态导出时客户端无法解析文案，会输出
// Next 默认的英文 404 页面。404 只有寥寥几行文案，因此只挑选所需的命名空间传下去
// （节省文档体积）。支持的 locale 与 web/src/i18n/config.ts 的 LOCALES 一致。
const MESSAGES: Record<Locale, Record<string, unknown>> = {
  zh: { notFound: (zhMessages as Record<string, unknown>).notFound },
};

// 访问不存在的 URL 时展示的页面。App Router 的 `not-found.tsx` 只会生成供
// `notFound()` 调用的内部路由（`/_not-found`），静态导出产出的顶层 `404.html`
// 并不使用它（所以导出的 404 曾是 Next 默认的英文页面）。
// `global-not-found.tsx` 必须 **直接返回完整的 HTML 文档**（它不继承根布局），
// 因此这里自行组织 html/body、字体变量、主题引导脚本与 messages。
// 通过 next.config.mjs 的 experimental.globalNotFound 开启。
export const metadata = {
  title: APP_CONFIG.meta.title,
  description: APP_CONFIG.meta.description,
};

export default function GlobalNotFound() {
  const { theme_mode, theme_preset, content_layout, navbar_style, sidebar_variant, sidebar_collapsible, font } =
    PREFERENCE_DEFAULTS;
  const locale = resolveLocale();
  return (
    <html
      lang={locale}
      data-theme-mode={theme_mode}
      data-theme-preset={theme_preset}
      data-content-layout={content_layout}
      data-navbar-style={navbar_style}
      data-sidebar-variant={sidebar_variant}
      data-sidebar-collapsible={sidebar_collapsible}
      data-font={font}
      suppressHydrationWarning
    >
      <head>
        {/* 在加载时应用主题与布局默认值，避免闪烁（与根布局一致）。 */}
        <ThemeBootScript />
      </head>
      <body className={`${fontVars} min-h-screen antialiased`}>
        <NextIntlClientProvider locale={locale} messages={MESSAGES[locale]}>
          <GlobalNotFoundBody />
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
