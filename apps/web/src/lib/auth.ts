import "server-only";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

export const sessionCookie = "__Host-schewise_session";
export const apiBaseURL = process.env.API_BASE_URL ?? "http://127.0.0.1:8080";

export type User = { id: string; email: string; role: string };

export async function requireUser(): Promise<User> {
  const token = (await cookies()).get(sessionCookie)?.value;
  if (!token) redirect("/login");
  const response = await fetch(`${apiBaseURL}/api/v1/auth/me`, {
    headers: { Cookie: `${sessionCookie}=${token}` },
    cache: "no-store",
    signal: AbortSignal.timeout(5000),
  });
  if (response.status === 401) redirect("/login");
  if (!response.ok) throw new Error("Authentication service is unavailable.");
  const data = (await response.json()) as { user: User };
  return data.user;
}
