import { BackendHealth } from "@/components/backend-health";
import { LogoutButton } from "@/components/logout-button";
import { requireUser } from "@/lib/auth";

export default async function Home() {
  const user = await requireUser();
  return (
    <main>
      <p className="eyebrow">Schewise / Development preview</p>
      <h1>A workspace for your database.</h1>
      <p className="intro">
        Open-source, AI-assisted database management. This starter connects the
        web app to the Go API. Database features are planned and are not
        available yet.
      </p>

      <section className="connection" aria-labelledby="account-heading">
        <h2 id="account-heading">Your account</h2>
        <p>
          {user.email} · {user.role}
        </p>
        <LogoutButton />
      </section>

      <BackendHealth />

      <section aria-labelledby="planned-capabilities">
        <h2 id="planned-capabilities">What we are building toward</h2>
        <ul>
          <li>Schema history, change review, and environment management.</li>
          <li>Mock datasets and database endpoints for API development.</li>
          <li>
            Schema mapping, data harmonization, and AI-assisted workflows.
          </li>
        </ul>
      </section>

      <footer>
        <a href="https://github.com/Thitipong-PP/schewise">
          Source code and contribution guide
        </a>
        <span>Apache-2.0</span>
      </footer>
    </main>
  );
}
