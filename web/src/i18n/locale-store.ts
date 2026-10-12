"use client";

import * as React from "react";

import { setClientCookie } from "@/lib/cookie.client";
import { getLocalStorageValue, setLocalStorageValue } from "@/lib/local-storage.client";

import { isLocale, LOCALE_COOKIE, LOCALE_STORAGE_KEY, type Locale, resolveLocale } from "./config";

// 用户选定的界面显示语言属于纯前端偏好，不存到服务端，只放在 localStorage(以及同名
// cookie)里。它不与账号同步，因此换浏览器或清除站点数据后会回到默认语言。agent 的
// 输出语言另行保存在服务端设置中。

function parseLocale(raw: string | null): Locale {
  return isLocale(raw) ? raw : resolveLocale();
}

// 同一标签页内的订阅者集合。storage 事件只在「其他」标签页触发，因此在当前标签页切换
// 语言后必须用 emit 通知本标签页的订阅者，才能立即重新渲染。
const listeners = new Set<() => void>();

function subscribe(listener: () => void) {
  listeners.add(listener);
  window.addEventListener("storage", listener);
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", listener);
  };
}

function emit() {
  for (const listener of listeners) listener();
}

function getSnapshot(): Locale {
  return parseLocale(getLocalStorageValue(LOCALE_STORAGE_KEY));
}

// 服务端没有 localStorage，所以先渲染构建默认 locale，hydrate 之后再让 getSnapshot
// 用用户选定的值纠正过来。
function getServerSnapshot(): Locale {
  return resolveLocale();
}

// 订阅当前显示语言的 hook。语言变化时，使用该 hook 的组件会重新渲染。
export function useActiveLocale(): Locale {
  return React.useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}

// 切换显示语言。写入 cookie 与 localStorage，并通知本标签页的订阅者。保留一年，
// 让下次访问仍记得这次选择。
export function setActiveLocale(locale: Locale) {
  if (!isLocale(locale)) return;
  setLocalStorageValue(LOCALE_STORAGE_KEY, locale);
  setClientCookie(LOCALE_COOKIE, locale, 365);
  emit();
}
