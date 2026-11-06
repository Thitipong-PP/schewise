// SPDX-License-Identifier: Apache-2.0
"use client";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { createContext, useContext, useEffect } from "react";
import {
  Database,
  GitBranch,
  Shield,
  ScanSearch,
  ScrollText,
  MailPlus,
  UserRound,
  UsersRound,
} from "lucide-react";
import { LogoutButton } from "@/components/logout-button";
import { api } from "@/lib/admin-api";
import type { Context } from "@/lib/admin-model";

const Access = createContext<Context | null>(null);
export const useAccess = () => useContext(Access);

export function AppShell({
  context,
  email,
  children,
}: {
  context: Context | null;
  email: string;
  children: React.ReactNode;
}) {
  const path = usePathname();
  const router = useRouter();
  useEffect(() => {
    const middleware = {
      onResponse({ response }: { response: Response }) {
        if (response.status === 401) {
          router.replace("/login");
          router.refresh();
        }
      },
    };
    api.use(middleware);
    return () => api.eject(middleware);
  }, [router]);
  const links = [
    {
      href: "/connections",
      label: "Connections",
      icon: Database,
      visible: context?.can_read_connections,
    },
    {
      href: "/admin/users",
      label: "Users",
      icon: UsersRound,
      visible: context?.root_permissions.includes("user:manage"),
    },
    { href: "/account", label: "Account", icon: UserRound, visible: true },
    {
      href: "/invites",
      label: "Invites",
      icon: MailPlus,
      UserRound,
      UsersRound,
      visible: context?.root_permissions.includes("user:manage"),
    },
    {
      href: "/",
      label: "Organization",
      icon: GitBranch,
      visible: context?.can_read_units,
    },
    {
      href: "/roles",
      label: "Roles",
      icon: Shield,
      visible: context?.root_permissions.includes("role:manage"),
    },
    {
      href: "/explain",
      label: "Access explorer",
      icon: ScanSearch,
      visible: context?.can_read_units,
    },
    {
      href: "/audit",
      label: "Audit log",
      icon: ScrollText,
      visible: context?.root_permissions.includes("audit:read"),
    },
  ];
  return (
    <Access.Provider value={context}>
      <div className="workspace">
        <header className="workspace-header">
          <Link href="/" className="workspace-brand">
            Schewise
          </Link>
          <div className="account-controls">
            <span title={email} className="account-email">
              {email}
            </span>
            <LogoutButton />
          </div>
        </header>
        <div className="workspace-layout">
          <nav className="workspace-nav" aria-label="Workspace">
            {links
              .filter((link) => link.visible)
              .map(({ href, label, icon: Icon }) => {
                const current =
                  href === "/"
                    ? path === "/" || path.startsWith("/units/")
                    : path.startsWith(href);
                return (
                  <Link
                    key={href}
                    href={href}
                    aria-current={current ? "page" : undefined}
                  >
                    <Icon size={17} aria-hidden="true" />
                    {label}
                  </Link>
                );
              })}
          </nav>
          <main className="workspace-main">
            {context || path.startsWith("/account") ? (
              children
            ) : (
              <>
                <h1>Access unavailable</h1>
                <p className="muted">
                  No workspace permissions are assigned to your account.
                </p>
              </>
            )}
          </main>
        </div>
      </div>
    </Access.Provider>
  );
}
