import { spawn } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { parseEnv } from "node:util";

const local = existsSync(".env") ? parseEnv(readFileSync(".env", "utf8")) : {};
const child = spawn(process.argv[2], process.argv.slice(3), {
  stdio: "inherit",
  env: { ...local, ...process.env },
});
child.on("error", () => {
  console.error("Could not start the requested development command.");
  process.exitCode = 1;
});
for (const signal of ["SIGINT", "SIGTERM"]) {
  process.on(signal, () => child.kill(signal));
}
child.on("exit", (code) => {
  process.exitCode = code ?? 1;
});
