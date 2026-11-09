// SPDX-License-Identifier: Apache-2.0
import { Connections } from "@/components/admin/connections";
export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return <Connections key={id} unitID={id} />;
}
