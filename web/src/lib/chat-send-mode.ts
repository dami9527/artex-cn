"use client";

import * as React from "react";

import { useTranslations } from "next-intl";

import { getLocalStorageValue, setLocalStorageValue } from "@/lib/local-storage.client";

// 会话输入框的发送/换行键位。纯前端偏好：只落 localStorage，不入库、不随账号同步，
// 因此换浏览器需要重设。见 issue #39——0.3.2 把 Ctrl+Enter 发送改成了 Enter 发送，
// 这里把旧键位还回来作为可选项。
export type ChatSendMode = "enter" | "ctrl-enter";

export const CHAT_SEND_MODE_KEY = "artex_chat_send_mode";
export const DEFAULT_CHAT_SEND_MODE: ChatSendMode = "enter";

// 设置页「发送方式」下拉使用的选项。标签随语言变化，因此不由模块常量提供，而是用
// 钩子返回（在纯模块里硬编码文案，会让其他 locale 显示成错误的语言）。value 是稳定
// 标识符，直接用于 localStorage 键与行为判定。
export function useChatSendModeOptions(): { value: ChatSendMode; label: string }[] {
  const t = useTranslations("chatSendMode.option");
  return [
    { value: "enter", label: t("enter") },
    { value: "ctrl-enter", label: t("ctrlEnter") },
  ];
}

function parseMode(raw: string | null): ChatSendMode {
  return raw === "ctrl-enter" || raw === "enter" ? raw : DEFAULT_CHAT_SEND_MODE;
}

// 同一标签页内的订阅者集合。localStorage 的 storage 事件只在「其他」标签页触发，
// 本页在设置里改完后要靠 emit 通知同页的输入框，否则得刷新才生效。
const listeners = new Set<() => void>();

function subscribe(listener: () => void) {
  listeners.add(listener);
  window.addEventListener("storage", listener);
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", listener);
  };
}

// 返回的是字符串字面量，Object.is 按值比较，因此 useSyncExternalStore 不会陷入
// 无限循环。
function getSnapshot(): ChatSendMode {
  return parseMode(getLocalStorageValue(CHAT_SEND_MODE_KEY));
}

// 服务端没有 localStorage，因此先渲染默认值，hydrate 后由 getSnapshot
// 把值纠正过来。
function getServerSnapshot(): ChatSendMode {
  return DEFAULT_CHAT_SEND_MODE;
}

export function useChatSendMode(): ChatSendMode {
  return React.useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}

export function setChatSendMode(mode: ChatSendMode) {
  setLocalStorageValue(CHAT_SEND_MODE_KEY, mode);
  for (const listener of listeners) listener();
}

// shouldSubmitOnKey 判断一次按键是否应当发送。
// isComposing / keyCode 229 是中文等输入法正在选字，必须放行，否则回车选词会误发送。
// enter 模式只排除 Shift，与 0.3.2 的行为逐字保持一致——不改设置的用户手感不变。
// ctrl-enter 模式同时接受 Ctrl 与 Cmd（macOS）。
export function shouldSubmitOnKey(e: React.KeyboardEvent, mode: ChatSendMode): boolean {
  if (e.key !== "Enter") return false;
  if (e.nativeEvent.isComposing || e.nativeEvent.keyCode === 229) return false;
  if (mode === "ctrl-enter") return e.ctrlKey || e.metaKey;
  return !e.shiftKey;
}
