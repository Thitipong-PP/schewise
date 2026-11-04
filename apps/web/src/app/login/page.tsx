import { LoginForm } from "@/components/login-form";

export default function LoginPage() {
  return (
    <main className="auth-page">
      <p className="eyebrow">Schewise</p>
      <h1>Sign in to your workspace.</h1>
      <p className="intro">
        Use the account provided by your administrator. Public registration is
        disabled.
      </p>
      <LoginForm />
    </main>
  );
}
