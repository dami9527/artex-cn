import { getRequestConfig } from "next-intl/server";

import { type Locale, resolveLocale } from "./config";

// 各 locale 的文案加载器。用显式 import 映射而不是动态路径，避免打包器把
// messages/ 整个目录当成 context 模块拖进来。
const loaders: Record<Locale, () => Promise<{ default: Record<string, unknown> }>> = {
  zh: () => import("../../messages/zh.json"),
};

// next-intl 的请求配置。这里不使用 i18n 路径路由（不占 URL 段、不加中间件），
// locale 直接在此决定。不引用 requestLocale，因此不会触发动态渲染。
export default getRequestConfig(async () => {
  const locale = resolveLocale();
  const messages = (await loaders[locale]()).default;
  return { locale, messages };
});
