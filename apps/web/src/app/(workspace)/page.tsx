// SPDX-License-Identifier: Apache-2.0
import { Organization } from "@/components/admin/organization";
import { accessContext } from "@/lib/access-server";
import { redirect } from "next/navigation";
export default async function Page() {
  const context = await accessContext();
  if (context && !context.can_read_units && context.can_read_connections)
    redirect("/connections");
  return <Organization />;
}
