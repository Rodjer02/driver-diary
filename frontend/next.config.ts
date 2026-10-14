import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  cacheComponents: true,
  partialPrefetching: true,
  // для Docker-образа на своём хосте; Vercel эту опцию игнорирует
  output: "standalone",
};

export default nextConfig;
