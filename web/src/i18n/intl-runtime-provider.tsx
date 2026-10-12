"use client";

import type { ReactNode } from "react";
import * as React from "react";

import { NextIntlClientProvider } from "next-intl";

import enMessages from "../../messages/en.json";
import esMessages from "../../messages/es.json";
import zhMessages from "../../messages/zh.json";
import type { Locale } from "./config";
import { useActiveLocale } from "./locale-store";

// 三种语言的文案全部静态打进包里。这样用户在设置界面切换语言时，无需重新构建、刷新
// 页面或异步加载，所有文案就能当场按所选语言重新渲染（与静态导出 output: "export"
// 也兼容）。
const MESSAGES: Record<Locale, Record<string, unknown>> = {
  en: enMessages as Record<string, unknown>,
  zh: zhMessages as Record<string, unknown>,
  es: esMessages as Record<string, unknown>,
};

// 运行环境(浏览器、预渲染服务器)的时区。
const TIME_ZONE = Intl.DateTimeFormat().resolvedOptions().timeZone;

// 负责运行期语言切换的 provider。它取代根布局里的 NextIntlClientProvider，
// 按用户选定的当前 locale(localStorage + 同名 cookie)挑选并注入 messages。
export function IntlRuntimeProvider({ children }: Readonly<{ children: ReactNode }>) {
  const locale = useActiveLocale();

  // 语言变化时同步 <html lang>。用客户端的选择覆盖根布局在服务端埋下的初始 lang，
  // 让屏幕阅读器与浏览器翻译按正确的语言工作。
  React.useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  // 与根布局的 provider 不同，这里不继承服务端请求配置，所以要显式传入 timeZone。
  // 漏掉的话 next-intl 会在预渲染期间报 ENVIRONMENT_FALLBACK 错误。
  return (
    <NextIntlClientProvider locale={locale} messages={MESSAGES[locale]} timeZone={TIME_ZONE}>
      {children}
    </NextIntlClientProvider>
  );
}
