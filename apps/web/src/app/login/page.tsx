import { displayFont, bodyFont } from "@/app/brand-fonts";
import Link from "next/link";
import { CodeXml, LockKeyhole } from "lucide-react";
import { LoginForm } from "@/components/login-form";
import "./login.css";

export default function LoginPage() {
  return (
    <div className={`login-shell ${displayFont.variable} ${bodyFont.variable}`}>
      <header className="login-header">
        <Link
          className="login-brand"
          href="/"
          prefetch={false}
          aria-label="Schewise home"
        >
          <span>Schewise</span>
        </Link>
        <a
          className="login-source"
          href="https://github.com/Thitipong-PP/schewise"
        >
          <CodeXml size={16} aria-hidden="true" />
          Source code
        </a>
      </header>

      <main className="login-main">
        <div className="login-content">
          <div className="login-identity">
            <span>Schewise</span>
          </div>
          <h1 className="login-heading" id="login-title">
            Sign in to your workspace.
          </h1>
          <LoginForm />
          <p className="login-access-note">
            <LockKeyhole size={14} aria-hidden="true" />
            Public registration is disabled.
          </p>
        </div>
      </main>

      <footer className="login-footer">
        <span>Self-hosted workspace</span>
        <span>Apache-2.0</span>
      </footer>
    </div>
  );
}
