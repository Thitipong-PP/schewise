// SPDX-License-Identifier: Apache-2.0
import { requireUser } from "@/lib/auth";
import { accessContext } from "@/lib/access-server";
import { AppShell } from "@/components/admin/shell";
import { displayFont, bodyFont } from "@/app/brand-fonts";
import "./workspace.css";

export default async function WorkspaceLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const user = await requireUser();
  const context = await accessContext();
  return (
    <div className={`${displayFont.variable} ${bodyFont.variable}`}>
      <AppShell context={context} email={user.email}>
        {children}
      </AppShell>
    </div>
  );
}
