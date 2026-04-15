DB_URL ?= postgres://yumyums:yumyums@localhost:5432/yumyums?sslmode=disable
PORT ?= 8080
STATIC_DIR ?= ..

.PHONY: dev build db-start db-stop db-reset db-seed seed

dev:
	STATIC_DIR=$(STATIC_DIR) PORT=$(PORT) DB_URL=$(DB_URL) SUPERADMIN_CONFIG=config/superadmins.yaml go run ./cmd/server

build:
	rm -rf cmd/server/public/*
	cp ../*.html ../*.js ../manifest.json cmd/server/public/
	cp -r ../icons cmd/server/public/ 2>/dev/null || true
	cp -r ../lib cmd/server/public/ 2>/dev/null || true
	go build -o bin/server ./cmd/server

db-start:
	docker run -d --name yumyums-pg \
	  -e POSTGRES_USER=yumyums \
	  -e POSTGRES_PASSWORD=yumyums \
	  -e POSTGRES_DB=yumyums \
	  -p 5432:5432 \
	  -v yumyums-pgdata:/var/lib/postgresql/data \
	  postgres:16

db-stop:
	docker stop yumyums-pg && docker rm yumyums-pg

db-reset: db-stop
	docker volume rm yumyums-pgdata
	$(MAKE) db-start

seed:
	DB_URL=$(DB_URL) go run ./cmd/seed

db-seed:
	@echo "Seeding hq_apps..."
	docker exec yumyums-pg psql -U yumyums -d yumyums -c " \
	  INSERT INTO hq_apps (slug, name, icon) VALUES \
	    ('purchasing', 'Purchasing', '🛒'), \
	    ('payroll', 'Payroll', '💰'), \
	    ('scheduling', 'Scheduling', '📅'), \
	    ('hiring', 'Hiring', '👥'), \
	    ('bi', 'BI', '📊'), \
	    ('users', 'Users', '🔐'), \
	    ('operations', 'Operations', '📋') \
	  ON CONFLICT (slug) DO NOTHING;"
	@echo "Seed complete."
