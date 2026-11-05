// SPDX-License-Identifier: Apache-2.0
import createClient from "openapi-fetch";
import type { paths } from "./api-types";

export const api = createClient<paths>({
  baseUrl: "",
  credentials: "same-origin",
});
export function result<T>(value: {
  data?: T;
  error?: unknown;
  response: Response;
}): T {
  if (!value.response.ok) {
    const error = value.error;
    throw new Error(
      error &&
        typeof error === "object" &&
        "error" in error &&
        typeof error.error === "string"
        ? error.error
        : "The request could not be completed.",
    );
  }
  return value.data as T;
}

export async function csrfHeaders(client: Pick<typeof api, "GET"> = api) {
  const data = result(
    await client.GET("/api/v1/auth/csrf", { cache: "no-store" }),
  );
  return { "X-CSRF-Token": data.csrf_token };
}

export function message(error: unknown) {
  return error instanceof Error
    ? error.message
    : "The service is unavailable. Please retry.";
}
