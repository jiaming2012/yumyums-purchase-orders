# ---------- Builder ----------
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src

# Cache deps first
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

# Backend source
COPY backend ./backend

# Frontend assets to embed (mirrors `task build` in backend/Taskfile.yml)
COPY *.html *.js manifest.json ./
COPY icons ./icons
COPY lib ./lib

RUN mkdir -p backend/cmd/server/public \
 && cp *.html *.js manifest.json backend/cmd/server/public/ \
 && cp -r icons backend/cmd/server/public/ \
 && cp -r lib backend/cmd/server/public/

RUN cd backend && CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags='-s -w' -o /out/server ./cmd/server

# ---------- Runtime ----------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=builder /out/server /app/server
COPY --from=builder /src/backend/config /app/config

USER app
EXPOSE 8080

ENV PORT=8080 \
    SUPERADMIN_CONFIG=/app/config/superadmins.yaml \
    TEMPLATE_CONFIG=/app/config/templates.yaml

ENTRYPOINT ["/app/server"]
