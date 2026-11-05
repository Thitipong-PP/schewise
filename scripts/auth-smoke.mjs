import assert from "node:assert/strict";

const base =
  process.env.SCHEWISE_SMOKE_BASE_URL ??
  process.env.SCHEWISE_ORIGIN ??
  "https://localhost:8443";
const origin = process.env.SCHEWISE_ORIGIN ?? "https://localhost:8443";
let cookie = "";
async function request(path, options = {}) {
  const response = await fetch(`${base}${path}`, {
    ...options,
    headers: { ...(cookie ? { Cookie: cookie } : {}), ...options.headers },
    redirect: "manual",
    signal: AbortSignal.timeout(15000),
  });
  for (const value of response.headers.getSetCookie()) {
    if (value.startsWith("__Host-schewise_session=")) {
      assert.match(value, /; Secure/i);
      assert.match(value, /; HttpOnly/i);
      assert.match(value, /; SameSite=Lax/i);
      cookie = value.split(";")[0];
    }
  }
  return response;
}

for (const path of ["/api/v1/health"]) {
  const response = await request(path);
  assert.equal(response.status, 200);
  assert.deepEqual(await response.json(), {
    status: "ok",
    service: "schewise-api",
  });
}
assert.equal((await request("/api/v1/health", { method: "POST" })).status, 405);
assert.equal((await request("/")).status, 307);
const loginPage = await request("/login");
assert.equal(loginPage.status, 200);
assert.match(await loginPage.text(), /Sign in/);
assert.equal((await request("/api/v1/auth/me")).status, 401);

const csrfResponse = await request("/api/v1/auth/csrf");
assert.equal(csrfResponse.status, 200);
const { csrf_token } = await csrfResponse.json();
const anonymousCookie = cookie;
const credentials = {
  email: process.env.SCHEWISE_ADMIN_EMAIL,
  password: process.env.SCHEWISE_ADMIN_PASSWORD,
};
assert.ok(
  credentials.email && credentials.password,
  "Set matching installation credentials for the smoke check",
);
assert.equal(
  (
    await request("/api/v1/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json", Origin: origin },
      body: JSON.stringify(credentials),
    })
  ).status,
  403,
);
const login = await request("/api/v1/auth/login", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    Origin: origin,
    "X-CSRF-Token": csrf_token,
  },
  body: JSON.stringify(credentials),
});
assert.equal(login.status, 200);
assert.notEqual(cookie, anonymousCookie);
assert.equal((await login.json()).user.role, "Administrator");
const dashboard = await request("/");
assert.equal(dashboard.status, 200);
assert.match(await dashboard.text(), /Organization/);
const authenticatedCSRF = await request("/api/v1/auth/csrf");
assert.equal(authenticatedCSRF.status, 200);
const nextCSRF = (await authenticatedCSRF.json()).csrf_token;
assert.equal(
  (
    await request("/api/v1/auth/logout", {
      method: "POST",
      headers: { Origin: origin, "X-CSRF-Token": nextCSRF },
    })
  ).status,
  204,
);
assert.equal((await request("/api/v1/auth/me")).status, 401);
assert.equal((await request("/")).status, 307);
console.log(
  "Auth smoke passed: secure cookie, CSRF, login, protected page, and logout.",
);
