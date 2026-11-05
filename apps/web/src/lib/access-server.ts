// SPDX-License-Identifier: Apache-2.0
import "server-only";
import { cookies } from "next/headers";
import createClient from "openapi-fetch";
import { apiBaseURL, sessionCookie } from "./auth";
import type { paths } from "./api-types";

export async function accessContext() {
  const token = (await cookies()).get(sessionCookie)?.value;
  const client = createClient<paths>({
    baseUrl: apiBaseURL,
    headers: { Cookie: `${sessionCookie}=${token ?? ""}` },
  });
  const response = await client.GET("/api/v1/access/context", {
    cache: "no-store",
    signal: AbortSignal.timeout(10000),
  });
  if (response.response.status === 403) return null;
  if (!response.data) throw new Error("Authorization service is unavailable.");
  return response.data;
}
