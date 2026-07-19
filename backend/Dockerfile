# Build context is the repo root (not backend/), so the build can reach
# ../*.html, ../*.js, etc. for the embedded frontend assets.
#
# Usage from repo root:
#   docker build -f backend/Dockerfile \
#     --build-arg GIT_SHA=$(git rev-parse --short HEAD) \
#     --build-arg BUILT_AT=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
#     -t yumyums-hq:latest .

# ─── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

ARG GIT_SHA=unknown
ARG BUILT_AT=unknown

RUN apk add --no-cache git

WORKDIR /src

# Cache deps separately from source.
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

# Frontend assets the binary embeds at /api/v1/* + static paths.
COPY *.html *.js manifest.json ./
COPY icons ./icons
COPY lib ./lib

# Backend source.
COPY backend ./backend

# Stage embedded assets where the Go embed directive expects them, then build.
RUN set -eux; \
    cd backend; \
    rm -rf cmd/server/public/*; \
    cp ../*.html ../*.js ../manifest.json cmd/server/public/; \
    cp -r ../icons cmd/server/public/; \
    cp -r ../lib cmd/server/public/; \
    CGO_ENABLED=0 go build \
      -ldflags "-X github.com/yumyums/hq/internal/version.GitSHA=${GIT_SHA} -X github.com/yumyums/hq/internal/version.BuiltAt=${BUILT_AT}" \
      -o /out/server ./cmd/server

# ─── Runtime stage ────────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 app

WORKDIR /app
COPY --from=builder /out/server /app/server
COPY backend/config /app/config

USER app
EXPOSE 8080

ENV PORT=8080 \
    BASE_URL=https://hq.yumyums.kitchen \
    SUPERADMIN_CONFIG=/app/config/superadmins.yaml

ENTRYPOINT ["/app/server"]
