DB_URL ?= postgres://yumyums:yumyums@localhost:5432/yumyums?sslmode=disable
PORT ?= 8080
STATIC_DIR ?= ..

.PHONY: dev build

dev:
	STATIC_DIR=$(STATIC_DIR) PORT=$(PORT) DB_URL=$(DB_URL) go run ./cmd/server

build:
	rm -rf cmd/server/public/*
	cp ../*.html ../*.js ../manifest.json cmd/server/public/
	cp -r ../icons cmd/server/public/ 2>/dev/null || true
	cp -r ../lib cmd/server/public/ 2>/dev/null || true
	go build -o bin/server ./cmd/server
