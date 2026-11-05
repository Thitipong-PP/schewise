import type { NextConfig } from "next";

// Rewrites are resolved at dev startup or production build time.
// Keep this server-side; never expose database credentials through NEXT_PUBLIC_*.
const apiBaseURL = (
  process.env.API_BASE_URL ?? "http://127.0.0.1:8080"
).replace(/\/+$/, "");

const nextConfig: NextConfig = {
  poweredByHeader: false,
  // Invitation/reset tokens are bearer secrets in paths; do not log incoming URLs.
  logging: { incomingRequests: false },
  async headers() {
    return [
      {
        source: "/:kind(invite|reset-password)/:path*",
        headers: [
          { key: "Referrer-Policy", value: "no-referrer" },
          { key: "Cache-Control", value: "no-store" },
          { key: "X-Robots-Tag", value: "noindex, nofollow" },
        ],
      },
    ];
  },
  async rewrites() {
    return [
      {
        source: "/api/v1/:path*",
        destination: `${apiBaseURL}/api/v1/:path*`,
      },
    ];
  },
};

export default nextConfig;
