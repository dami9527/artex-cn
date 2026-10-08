"use client";

import Link from "next/link";

import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";

// global-not-found 의 본문. 문자열 해석에 훅이 필요해 클라이언트 경계로 분리했다
// (global-not-found.tsx 자체는 전체 HTML 문서를 반환해야 해서 서버 컴포넌트로 둔다).
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
