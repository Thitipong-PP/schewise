// SPDX-License-Identifier: Apache-2.0
import { SchemaBrowser } from "@/components/admin/schema-browser";
export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return <SchemaBrowser key={id} id={id} />;
}
