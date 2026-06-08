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
# VERSION is the tag this image will be pushed under (computed by
# scripts/version.sh in CI). It's baked into the bundle as
# NEXT_PUBLIC_WEB_VERSION so the running app can show its own version
# without a separate runtime lookup. Required — fail the build rather
# than silently shipping the stale package.json fallback when CI forgets
# to pass it. For local builds, pass e.g. `--build-arg VERSION=dev`.
#
# GIT_COMMIT/GIT_TREE_STATE/BUILD_DATE mirror the apiserver's version payload
# (api/v1/version). Git isn't available in this build context (only ./web is
# copied), so CI computes them and passes them in; they're optional and
# default to empty for local builds.
ARG VERSION
ARG GIT_COMMIT=""
ARG GIT_TREE_STATE=""
ARG BUILD_DATE=""
ENV NEXT_TELEMETRY_DISABLED=1 \
    NEXT_PUBLIC_WEB_VERSION=${VERSION} \
    NEXT_PUBLIC_WEB_GIT_COMMIT=${GIT_COMMIT} \
    NEXT_PUBLIC_WEB_GIT_TREE_STATE=${GIT_TREE_STATE} \
    NEXT_PUBLIC_WEB_BUILD_DATE=${BUILD_DATE}
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN test -n "${VERSION}" || (echo "ERROR: VERSION build-arg is required (pass --build-arg VERSION=...)" >&2 && exit 1) \
 && npm run build

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
