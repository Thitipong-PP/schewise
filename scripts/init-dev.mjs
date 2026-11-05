import { randomBytes } from "node:crypto";
import { writeFileSync } from "node:fs";

// This command never overwrites installation credentials or prints passwords.
try {
  writeFileSync(
    ".env",
    `# Local installation secrets. Never commit this file.
SCHEWISE_ORIGIN=https://localhost:8443
PUBLIC_BASE_URL=https://localhost:8443
SCHEWISE_INVITE_TTL=168h
SCHEWISE_ADMIN_EMAIL=admin@example.test
SCHEWISE_ADMIN_PASSWORD=${randomBytes(24).toString("hex")}
SCHEWISE_SESSION_TTL=24h
SCHEWISE_STORE_SESSION_IP=true
SCHEWISE_PASSWORD_RESET_TTL=24h
PGHOST=127.0.0.1
PGPORT=5432
PGDATABASE=schewise
PGUSER=schewise
PGPASSWORD=${randomBytes(24).toString("hex")}
POSTGRES_PASSWORD=${randomBytes(24).toString("hex")}
`,
    { flag: "wx", mode: 0o600 },
  );
  console.log(
    "Created .env with unique local credentials. Review it before starting the app.",
  );
} catch (error) {
  if (error.code !== "EEXIST") throw error;
  console.log(".env already exists; installation credentials were preserved.");
}
