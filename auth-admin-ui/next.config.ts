import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  async headers() {
    return [
      {
        // Prevent CDN from caching HTML pages — only static assets (_next/static/)
        // should be cached long-term (they use content-hash filenames).
        // Stale HTML pointing to old chunk URLs causes "Application error: a
        // client-side exception has occurred" after redeployment.
        source: "/((?!_next/static|_next/image|favicon\\.ico).*)",
        headers: [
          { key: "Cache-Control", value: "no-store, must-revalidate" },
        ],
      },
    ];
  },
};

export default nextConfig;
