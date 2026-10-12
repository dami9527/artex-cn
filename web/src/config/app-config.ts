import { DEFAULT_LOCALE, type Locale, resolveLocale } from "@/i18n/config";

import packageJson from "../../package.json";

const currentYear = new Date().getFullYear();

// 浏览器标签页标题与搜索引擎元描述。作为 layout.tsx 的静态 metadata 注入所有页面的
// <title>·<meta name="description">，因此按构建生效的 locale 分三种语言取值。
// 涉及搜索排名的静态文案统一用冒号写成「产品名: 说明」的形式，不用破折号。
// 这份 meta 是构建期静态值，不响应运行期语言切换（只是标签页标题一行，可以接受）。
const META_BY_LOCALE: Record<Locale, { title: string; description: string }> = {
  en: {
    title: "ARTEX: Autonomous Penetration Testing Console",
    description: "An LLM-driven system console for autonomous penetration testing",
  },
  zh: {
    title: "ARTEX: 自主渗透测试控制台",
    description: "LLM 驱动的自主渗透测试系统控制台",
  },
  es: {
    title: "ARTEX: Consola de Pruebas de Penetración Autónomas",
    description: "Una consola de sistema impulsada por LLM para pruebas de penetración autónomas",
  },
};

export const APP_CONFIG = {
  name: "ARTEX",
  version: packageJson.version,
  copyright: `© ${currentYear}, ARTEX.`,
  meta: META_BY_LOCALE[resolveLocale()] ?? META_BY_LOCALE[DEFAULT_LOCALE],
};
