import { NextResponse, type NextRequest } from "next/server";

const cookieName = "__Host-schewise_session";

// Next.js 16 calls this convention `proxy.ts` (formerly middleware.ts).
// Go performs authoritative session and permission checks on every protected API.
export async function proxy(request: NextRequest) {
  const token = request.cookies.get(cookieName)?.value;
  const origin = process.env.SCHEWISE_ORIGIN ?? "https://localhost:8443";
  const login = () => {
    const destination = new URL("/login", origin);
    destination.searchParams.set("next", request.nextUrl.pathname);
    const response = NextResponse.redirect(destination);
    response.headers.set("Cache-Control", "no-store");
    return response;
  };
  if (!token) return login();
  try {
    const response = await fetch(
      `${process.env.API_BASE_URL ?? "http://127.0.0.1:8080"}/api/v1/auth/me`,
      {
        headers: { Cookie: `${cookieName}=${token}` },
        cache: "no-store",
        signal: AbortSignal.timeout(5000),
      },
    );
    if (response.status === 401) return login();
    if (!response.ok) throw new Error("Authentication unavailable");
    const next = NextResponse.next();
    next.headers.set("Cache-Control", "no-store");
    return next;
  } catch {
    return new NextResponse(
      "Authentication service is temporarily unavailable.",
      {
        status: 503,
        headers: { "Cache-Control": "no-store" },
      },
    );
  }
}

export const config = {
  matcher: [
    "/((?!api|login|invite/|reset-password/|_next/static|_next/image|favicon.ico).*)",
  ],
};
