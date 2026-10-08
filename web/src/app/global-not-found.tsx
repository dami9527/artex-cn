import { NextIntlClientProvider } from "next-intl";

import { APP_CONFIG } from "@/config/app-config";
import { type Locale, resolveLocale } from "@/i18n/config";
import { fontVars } from "@/lib/fonts/registry";
import { PREFERENCE_DEFAULTS } from "@/lib/preferences/preferences-config";
import { ThemeBootScript } from "@/scripts/theme-boot";

import koMessages from "../../messages/ko.json";
import zhMessages from "../../messages/zh.json";
import { GlobalNotFoundBody } from "./_components/global-not-found-body";

import "./globals.css";

// 이 페이지는 루트 레이아웃을 상속하지 않으므로(전체 HTML 문서를 직접 반환한다)
// NextIntlClientProvider 에 **메시지를 직접 넘겨야** 한다. 넘기지 않으면 정적
// 내보내기 시점에 클라이언트가 문구를 해석하지 못해 Next 기본 영어 404 가 나간다.
// 404 는 문구 몇 줄뿐이라 필요한 네임스페이스만 추려서 넘긴다(문서 크기 절약).
// 지원 locale 은 web/src/i18n/config.ts 의 LOCALES(ko·zh)와 같다.
const MESSAGES: Record<Locale, Record<string, unknown>> = {
  ko: { notFound: (koMessages as Record<string, unknown>).notFound },
  zh: { notFound: (zhMessages as Record<string, unknown>).notFound },
};

// 존재하지 않는 URL 로 들어왔을 때 보여줄 화면. App Router 의 `not-found.tsx` 는
// `notFound()` 호출용 내부 라우트(`/_not-found`)를 만들 뿐, 정적 내보내기가 만드는
// 최상위 `404.html` 에는 쓰이지 않는다(그래서 내보낸 404 가 Next 기본 영어 페이지였다).
// `global-not-found.tsx` 는 **전체 HTML 문서를 직접 반환해야** 하므로(루트 레이아웃을
// 상속하지 않는다) 여기서 html/body·글꼴 변수·테마 부트스트랩·메시지를 직접 구성한다.
// next.config.mjs 의 experimental.globalNotFound 로 켠다.
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
        {/* 테마·레이아웃 기본값을 로드 시점에 적용해 깜빡임을 막는다(루트 레이아웃과 동일). */}
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
