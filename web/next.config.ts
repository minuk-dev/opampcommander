import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  // `standalone` produces a self-contained server bundle in
  // `.next/standalone/`. The Docker image only needs Node.js + that
  // directory + the static assets — no npm install at runtime.
  output: 'standalone',
};

export default nextConfig;
