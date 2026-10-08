import { resolveLocale } from "@/i18n/config";

import packageJson from "../../package.json";

const currentYear = new Date().getFullYear();

// 浏览器标签页标题与搜索引擎元描述。作为 layout.tsx 的静态 metadata 注入所有
// 页面的 <title>·<meta name="description">，因此按生效的 locale 取值。
// 本仓库只有 zh 一种语言，取值结果固定为下面的中文文案。
const META_BY_LOCALE = {
  zh: {
    title: "ARTEX — 自主渗透测试控制台",
    description: "LLM 驱动的自主渗透测试系统控制台",
  },
} as const;

export const APP_CONFIG = {
  name: "ARTEX",
  version: packageJson.version,
  copyright: `© ${currentYear}, ARTEX.`,
  meta: META_BY_LOCALE[resolveLocale()],
};
