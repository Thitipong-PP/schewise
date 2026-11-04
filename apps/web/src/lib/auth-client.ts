export async function authMutation(
  path: string,
  input?: { email: string; password: string },
) {
  const csrfResponse = await fetch("/api/v1/auth/csrf", {
    cache: "no-store",
    signal: AbortSignal.timeout(10000),
  });
  if (!csrfResponse.ok)
    throw new Error("Could not initialize a secure session.");
  const { csrf_token } = (await csrfResponse.json()) as { csrf_token: string };
  const response = await fetch(`/api/v1/auth/${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf_token },
    body: input ? JSON.stringify(input) : undefined,
    cache: "no-store",
    signal: AbortSignal.timeout(15000),
  });
  if (!response.ok) {
    const result = (await response.json()) as { error?: string };
    throw new Error(result.error ?? "The request could not be completed.");
  }
}
