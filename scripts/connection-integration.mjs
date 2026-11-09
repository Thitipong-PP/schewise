// SPDX-License-Identifier: Apache-2.0
import { spawnSync } from "node:child_process";
const env = { ...process.env, SCHEWISE_INTEGRATION_TESTS: "1" };
if (!env.SCHEWISE_TEST_TARGET_ADMIN_URL && !env.SCHEWISE_TEST_DATABASE_URL) {
  const url = new URL("postgres://127.0.0.1:5433/postgres?sslmode=disable");
  url.username = "postgres";
  url.password =
    env.DEMO_POSTGRES_PASSWORD || "synthetic-demo-postgres-password";
  env.SCHEWISE_TEST_TARGET_ADMIN_URL = url.toString();
}
const child = spawnSync("go", ["test", "-race", "-count=1", "./..."], {
  env,
  stdio: "inherit",
});
process.exit(child.status ?? 1);
