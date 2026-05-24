# syntax=docker/dockerfile:1.7

# Multi-stage build for the Next.js standalone output. Final image carries
# only Node.js + the prebuilt server, no source / no npm install.

ARG NODE_VERSION=20-alpine

# --- deps: install only what the build needs ---------------------------------
FROM node:${NODE_VERSION} AS deps
WORKDIR /app
COPY package.json package-lock.json ./
# Build-time toolchains for native deps. None today, but keeps it stable if
# a transitive dep needs node-gyp later.
RUN apk add --no-cache libc6-compat && npm ci --no-audit --no-fund

# --- builder: produce .next/standalone ---------------------------------------
FROM node:${NODE_VERSION} AS builder
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

# --- runtime: minimal, non-root ---------------------------------------------
FROM node:${NODE_VERSION} AS runner
WORKDIR /app
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0

# Non-root user for runtime
RUN addgroup --system --gid 1001 nodejs \
 && adduser  --system --uid 1001 --ingroup nodejs nextjs

# Standalone server bundle. The Next.js build emits a self-contained
# `server.js` plus a trimmed `node_modules`.
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
COPY --from=builder --chown=nextjs:nodejs /app/public ./public

USER nextjs
EXPOSE 3000

# The server reads OPAMP_API_URL at request time (server-side proxy route),
# so it can be overridden per environment without rebuilding the image.
CMD ["node", "server.js"]
