// SPDX-License-Identifier: Apache-2.0
import { UnitDetail } from "@/components/admin/unit-detail";
export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return <UnitDetail key={id} id={id} />;
}
