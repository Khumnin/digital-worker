# syntax=docker/dockerfile:1

# ── Build stage — runs on the host platform (arm64 on Apple Silicon) ──────────
# Next.js output is platform-agnostic JavaScript, so building natively avoids
# the QEMU SIGSEGV issue with Node.js in cross-compile emulation.
FROM --platform=$BUILDPLATFORM node:22-alpine AS builder
WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .

ARG NEXT_PUBLIC_API_URL=https://auth.tgstack.dev
ARG NEXT_PUBLIC_APP_NAME="TGX Auth Console"
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_APP_NAME=$NEXT_PUBLIC_APP_NAME

RUN npm run build

# ── Production stage — targets the deployment platform (linux/amd64) ─────────
FROM --platform=$TARGETPLATFORM node:22-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production

RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]
