import type { NextConfig } from "next";

// Rewrites are resolved at dev startup or production build time.
// Keep this server-side; never expose database credentials through NEXT_PUBLIC_*.
const apiBaseURL = (
  process.env.API_BASE_URL ?? "http://127.0.0.1:8080"
).replace(/\/+$/, "");

const nextConfig: NextConfig = {
  poweredByHeader: false,
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
