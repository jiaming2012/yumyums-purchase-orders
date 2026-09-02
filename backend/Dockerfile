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

# ── The RxDB sync client layer. Ledger T-23 decision 59 AND ITS TRAP. ────────
#
# 🛑 THESE THREE LINES ARE INDIVISIBLE FROM build-sw.js's globPatterns ENTRIES.
# `workflows.html` now carries <script type="module" src="sync-rxdb/bootstrap.js">,
# which statically imports sync-rxdb/client.js, which imports BOTH
# sync-schema/collections.js AND vendor/rxdb.bundle.js — so all three are
# precached. A precached URL that 404s fails the ENTIRE service-worker install
# for every returning client, and it presents as "the PWA stopped updating" with
# no visible cause. Until this card, `vendor/` was NEVER copied into the image.
#
# NARROW ON PURPOSE, not whole directories: vendor/ also holds the generator's
# own inputs (build-vendor.sh, package.json, package-lock.json, src/*.mjs) and
# sync-schema/ holds SQL DDL, none of which the browser loads and none of which
# should be publicly served. tests/sw-manifest.spec.js asserts, for EVERY
# precache entry, that it is covered by a copy here — so a future
# vendor/<sub>/foo.bundle.js that the glob picks up and this line misses reds
# the suite instead of bricking the service worker.
COPY vendor/rxdb.bundle.js ./vendor/
COPY sync-rxdb/*.js ./sync-rxdb/
COPY sync-schema/collections.js ./sync-schema/

# Backend source.
COPY backend ./backend

# Stage embedded assets where the Go embed directive expects them, then build.
# version.json is a git-ignored build artifact (build-sw.js writes it locally),
# so it must be generated here from the authoritative Frontend constant —
# sw.js PRECACHES it, and a 404 aborts the whole service-worker install,
# pinning every returning client to the previous frontend.
#
# 🛑 THE BYTES MUST MATCH scripts/write-version-json.js EXACTLY — trailing '\n'
# INCLUDED. That module (the authoritative payload definition, required by
# build-sw.js) writes `JSON.stringify({frontend}) + '\n'` = `{"frontend":"X"}\n`.
# sw.js's precache REVISION for version.json is hashed against THAT copy, so if
# this printf emits different bytes (e.g. drops the newline) the manifest
# revision describes a file the image does not serve, and the served bytes and
# the hashed bytes diverge in prod — B-135. `tests/version-json-parity.spec.js`
# reads this printf's format literal and asserts byte-identity; drop the `\n`
# and it reds. The `\n` at the end of the format below is that byte.
RUN set -eux; \
    cd backend; \
    rm -rf cmd/server/public/*; \
    cp ../*.html ../*.js ../manifest.json cmd/server/public/; \
    fe_ver="$(sed -n 's/.*Frontend *= *"\([^"]*\)".*/\1/p' internal/version/version.go)"; \
    test -n "$fe_ver"; \
    printf '{"frontend":"%s"}\n' "$fe_ver" > cmd/server/public/version.json; \
    cp -r ../icons cmd/server/public/; \
    cp -r ../lib cmd/server/public/; \
    cp -r ../vendor cmd/server/public/; \
    cp -r ../sync-rxdb cmd/server/public/; \
    cp -r ../sync-schema cmd/server/public/; \
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
