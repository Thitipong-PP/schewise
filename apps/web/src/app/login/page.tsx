import localFont from "next/font/local";
import Link from "next/link";
import { LoginForm } from "@/components/login-form";
import { WorkspaceIllustration } from "./workspace-illustration";
import "./login.css";

const displayFont = localFont({
  src: "./fonts/host-grotesk.ttf",
  variable: "--font-login-display",
  weight: "300 800",
  display: "swap",
});

const bodyFont = localFont({
  src: "./fonts/instrument-sans.ttf",
  variable: "--font-login-body",
  weight: "400 700",
  display: "swap",
});

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
          <span className="login-brand-mark" aria-hidden="true">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
              <path
                d="M7 5v10a4 4 0 0 0 4 4h6M7 9h6a4 4 0 0 0 4-4"
                stroke="currentColor"
                strokeWidth="1.8"
              />
              <circle cx="7" cy="5" r="2.5" fill="currentColor" />
              <circle cx="17" cy="5" r="2.5" fill="currentColor" />
              <circle cx="17" cy="19" r="2.5" fill="currentColor" />
            </svg>
          </span>
          Schewise
        </Link>
        <a
          className="login-source"
          href="https://github.com/Thitipong-PP/schewise"
        >
          Open source
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            aria-hidden="true"
          >
            <path
              d="M6 18 18 6M6 6h12v12"
              stroke="currentColor"
              strokeWidth="1.6"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </a>
      </header>

      <main className="login-main">
        <div className="login-intro">
          <p className="login-eyebrow">
            <span aria-hidden="true" /> Your database workspace
          </p>
          <h1>Sign in to your workspace.</h1>
          <p className="login-description">
            Use the account provided by your administrator.
          </p>
          <WorkspaceIllustration />
        </div>
        <div className="login-card">
          <LoginForm />
          <p className="login-access-note">
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              aria-hidden="true"
            >
              <rect
                x="5"
                y="10"
                width="14"
                height="11"
                rx="2"
                stroke="currentColor"
                strokeWidth="1.6"
              />
              <path
                d="M8 10V7a4 4 0 0 1 8 0v3m-4 5v2"
                stroke="currentColor"
                strokeWidth="1.6"
                strokeLinecap="round"
              />
            </svg>
            Public registration is disabled.
          </p>
        </div>
      </main>

      <footer className="login-footer">
        <span>A workspace for your database.</span>
        <span>Apache-2.0</span>
      </footer>
    </div>
  );
}
