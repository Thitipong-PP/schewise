import assert from "node:assert/strict";
import test from "node:test";
import { ESLint } from "eslint";

// Exercise real plugin rule execution during upgrades, not just config loading.
// The legacy plugins' published peer ranges currently stop at ESLint 9.
test("ESLint executes TypeScript, React, accessibility, import, and Next rules", async () => {
  const eslint = new ESLint();
  const [result] = await eslint.lintText(
    `
const values: any[] = ['one'];
const Invalid = () => <section><img src="/fixture.png" />{values.map(value => <span>{value}</span>)}</section>;
export default { Invalid };
`,
    { filePath: "src/toolchain-probe.tsx" },
  );
  const reported = new Set(result.messages.map(({ ruleId }) => ruleId));
  for (const rule of [
    "@typescript-eslint/no-explicit-any",
    "react/jsx-key",
    "jsx-a11y/alt-text",
    "import/no-anonymous-default-export",
    "@next/next/no-img-element",
  ]) {
    assert.ok(reported.has(rule), `Expected ${rule} to execute and report`);
  }
  assert.ok(result.messages.every(({ fatal }) => !fatal));
});

test("ESLint accepts valid TypeScript and JSX with the same plugin configuration", async () => {
  const eslint = new ESLint();
  const [result] = await eslint.lintText(
    `
const values: string[] = ['one'];
export default function Valid() {
  return <section>{values.map(value => <span key={value}>{value}</span>)}</section>;
}
`,
    { filePath: "src/toolchain-probe.tsx" },
  );
  assert.deepEqual(result.messages, []);
});
