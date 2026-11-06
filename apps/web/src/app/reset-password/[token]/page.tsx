// SPDX-License-Identifier: Apache-2.0
import type { Metadata } from "next";
import Link from "next/link";
import { displayFont, bodyFont } from "@/app/brand-fonts";
import { ResetPasswordForm } from "@/components/reset-password-form";
import "@/app/login/login.css";

export const dynamic = "force-dynamic";
export const metadata: Metadata = {
  title: "Reset password · Schewise",
  referrer: "no-referrer",
  robots: { index: false, follow: false },
};

export default async function Page({
  params,
}: {
  params: Promise<{ token: string }>;
}) {
  const { token } = await params;
  return (
    <div className={`login-shell ${displayFont.variable} ${bodyFont.variable}`}>
      <header className="login-header">
        <Link href="/login" className="login-brand" prefetch={false}>
          Schewise
        </Link>
      </header>
      <main className="login-main">
        <div className="login-content">
          <div className="login-identity">Schewise</div>
          <h1 className="login-heading" id="reset-title">
            Reset your password.
          </h1>
          <ResetPasswordForm token={token} />
        </div>
      </main>
      <footer className="login-footer">
        <span>Self-hosted workspace</span>
        <span>Apache-2.0</span>
      </footer>
    </div>
  );
}
