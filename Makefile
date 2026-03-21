include .env
export
COMPOSE_INFRA    = docker compose -f $(CURDIR)/docker-compose.yml
COMPOSE_ALL      = docker compose -f $(CURDIR)/docker-compose.yml -f $(CURDIR)/deployments/docker-compose.services.yml

# ── Infrastructure ────────────────────────────────────────────────────────────

infra:
	$(COMPOSE_INFRA) up -d

infra-down:
	$(COMPOSE_INFRA) down

infra-down-hard:
	$(COMPOSE_INFRA) down -v

# ── All services ──────────────────────────────────────────────────────────────

up:
	$(COMPOSE_ALL) up -d

rebuild:
	$(COMPOSE_ALL) up --build -d

down:
	$(COMPOSE_ALL) down

down-hard:
	$(COMPOSE_ALL) down -v

# ── Logs ──────────────────────────────────────────────────────────────────────

logs:
	$(COMPOSE_ALL) logs -f

logs-api:
	$(COMPOSE_ALL) logs -f api-service

logs-device:
	$(COMPOSE_ALL) logs -f device-service

logs-simulator:
	$(COMPOSE_ALL) logs -f simulator-service

logs-postgres:
	$(COMPOSE_INFRA) logs -f postgres

logs-redis:
	$(COMPOSE_INFRA) logs -f redis

logs-mqtt:
	$(COMPOSE_INFRA) logs -f mosquitto

# ── Individual service rebuild ────────────────────────────────────────────────

rebuild-api:
	$(COMPOSE_ALL) up --build -d api-service

rebuild-device:
	$(COMPOSE_ALL) up --build -d device-service

rebuild-simulator:
	$(COMPOSE_ALL) up --build -d simulator-service

# ── Shell access ──────────────────────────────────────────────────────────────

shell-api:
	$(COMPOSE_ALL) exec api-service sh

shell-device:
	$(COMPOSE_ALL) exec device-service sh

shell-simulator:
	$(COMPOSE_ALL) exec simulator-service sh

shell-postgres:
	$(COMPOSE_INFRA) exec postgres psql -U $${POSTGRES_USER} -d $${POSTGRES_DB}

shell-timescale:
	$(COMPOSE_INFRA) exec timescaledb psql -U $${TIMESCALE_USER} -d $${TIMESCALE_DB}

shell-redis:
	$(COMPOSE_INFRA) exec redis redis-cli -a $${REDIS_PASSWORD}

# ── Status ────────────────────────────────────────────────────────────────────

ps:
	$(COMPOSE_ALL) ps