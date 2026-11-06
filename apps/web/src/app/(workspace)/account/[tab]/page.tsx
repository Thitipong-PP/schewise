// SPDX-License-Identifier: Apache-2.0
import { notFound } from "next/navigation";
import { Account, type AccountTab } from "@/components/account/account";
export default async function Page({
  params,
}: {
  params: Promise<{ tab: string }>;
}) {
  const { tab } = await params;
  if (!["profile", "security", "sessions", "access"].includes(tab)) notFound();
  return <Account tab={tab as AccountTab} />;
}
