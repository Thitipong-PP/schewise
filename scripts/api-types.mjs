// SPDX-License-Identifier: Apache-2.0

import { readFile, writeFile } from "node:fs/promises";
import openapiTS, { astToString } from "openapi-typescript";
import prettier from "prettier";

const schema = new URL("../docs/openapi.json", import.meta.url);
const target = new URL("../apps/web/src/lib/api-types.ts", import.meta.url);
const nodes = await openapiTS(schema);
const output = await prettier.format(
  "// SPDX-License-Identifier: Apache-2.0\n\n// Generated from backend DTOs and routes. Run make api-generate; do not edit.\n\n" +
    astToString(nodes),
  { parser: "typescript" },
);
if (process.argv.includes("--check")) {
  if ((await readFile(target, "utf8")) !== output) {
    console.error("API types are stale. Run make api-generate.");
    process.exitCode = 1;
  } else {
    console.log("TypeScript API types match the backend OpenAPI contract.");
  }
} else {
  await writeFile(target, output);
}
