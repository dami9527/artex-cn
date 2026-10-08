"use client";

import Link from "next/link";

import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";

// global-not-found 的正文。解析文案需要 hook，因此拆成客户端边界
// （global-not-found.tsx 本身要返回完整 HTML 文档，只能保持为服务端组件）。
export function GlobalNotFoundBody() {
  const t = useTranslations("notFound");
  return (
    <div className="flex h-dvh flex-col items-center justify-center space-y-2 text-center">
      <h1 className="font-semibold text-2xl">{t("title")}</h1>
      <p className="text-muted-foreground">{t("description")}</p>
      <Link prefetch={false} replace href="/function/tasks">
        <Button variant="outline">{t("back")}</Button>
      </Link>
    </div>
  );
}
