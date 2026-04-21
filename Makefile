include .env
export

COMPOSE_INFRA = docker compose -f $(CURDIR)/docker-compose.yml
COMPOSE_ALL   = docker compose -f $(CURDIR)/docker-compose.yml -f $(CURDIR)/deployments/docker-compose.services.yml

SIMULATOR_CONTAINER = climate-control-simulator-service-1
CURRENT_SIM         = $(shell docker inspect $(SIMULATOR_CONTAINER) \
                        --format "{{range .Args}}{{.}} {{end}}" 2>/dev/null | \
                        grep -o '\-\-simulation=[^ ]*' | cut -d= -f2)

# ── Project lifecycle ─────────────────────────────────────────────────────────

up:
	$(COMPOSE_ALL) up -d

down:
	$(COMPOSE_ALL) down

down-hard:
	$(COMPOSE_ALL) down -v

rebuild:
	$(COMPOSE_ALL) up --build -d

# ── Infrastructure ────────────────────────────────────────────────────────────

infra:
	$(COMPOSE_INFRA) up -d

infra-down:
	$(COMPOSE_INFRA) down

infra-down-hard:
	$(COMPOSE_INFRA) down -v

# ── Service rebuild ───────────────────────────────────────────────────────────

rebuild-api:
	$(COMPOSE_ALL) up --build -d api-service

rebuild-device:
	$(COMPOSE_ALL) up --build -d device-service

restart-device:
	$(COMPOSE_ALL) restart device-service

rebuild-simulator:
	$(COMPOSE_ALL) up --build -d simulator-service

# ── Debug ─────────────────────────────────────────────────────────────────────

# Set device-service debug level and restart to apply
# Usage: make debug-device-info / debug-device-verbose / debug-device-off
#        make debug-device-trace-ingestion-on/off
#        make debug-device-trace-tick-on/off
debug-device-info:
	sed -i 's/^DEVICE_DEBUG=.*/DEVICE_DEBUG=info/' .env
	DEVICE_DEBUG=info $(COMPOSE_ALL) up -d device-service

debug-device-verbose:
	sed -i 's/^DEVICE_DEBUG=.*/DEVICE_DEBUG=verbose/' .env
	DEVICE_DEBUG=verbose $(COMPOSE_ALL) up -d device-service

debug-device-off:
	sed -i 's/^DEVICE_DEBUG=.*/DEVICE_DEBUG=/' .env
	DEVICE_DEBUG= $(COMPOSE_ALL) up -d device-service

debug-device-trace-ingestion-on:
	sed -i 's/^DEVICE_TRACE_INGESTION=.*/DEVICE_TRACE_INGESTION=true/' .env

debug-device-trace-ingestion-off:
	sed -i 's/^DEVICE_TRACE_INGESTION=.*/DEVICE_TRACE_INGESTION=false/' .env

debug-device-trace-tick-on:
	sed -i 's/^DEVICE_TRACE_TICK=.*/DEVICE_TRACE_TICK=true/' .env

debug-device-trace-tick-off:
	sed -i 's/^DEVICE_TRACE_TICK=.*/DEVICE_TRACE_TICK=false/' .env

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

# ── Go ────────────────────────────────────────────────────────────────────────

go-vet:
	go vet ./api-service/... ./device-service/... ./shared/... ./simulator-service/...

go-build:
	go build ./api-service/... ./device-service/... ./shared/... ./simulator-service/...

# ── API tests ─────────────────────────────────────────────────────────────────

# Full integration suite — requires a fresh database
# Usage: make down-hard && make up && make test-api-integration
test-api-integration:
	newman run tests/postman/climate-control-integration.collection.json \
		-e tests/postman/integration.environment.json \
		--delay-request 100

# Smoke suite — repeatable, safe to run against a live database
test-api-smoke:
	newman run tests/postman/climate-control-smoke.collection.json \
		-e tests/postman/smoke.environment.json \
		--delay-request 100

# ── Simulator ─────────────────────────────────────────────────────────────────

# Start simulator with given simulation file — creates a fresh container
# picking up any rebuilt images. Stops existing container first if running.
# Usage: make simulator-start SIM=default
simulator-start:
	$(COMPOSE_ALL) stop simulator-service 2>/dev/null || true
	SIMULATOR_SIMULATION=$(or $(SIM),default) $(COMPOSE_ALL) up -d simulator-service

# Stop running simulator — data preserved in DB
simulator-stop:
	$(COMPOSE_ALL) stop simulator-service

# Restart the stopped container — reuses existing image, re-runs provisioning
# Use when you have not rebuilt the image and just want to restart after a stop
simulator-resume:
	$(COMPOSE_ALL) start simulator-service

# Show current simulation file and container status
simulator-status:
	@echo "Simulation: $(or $(CURRENT_SIM),none)"
	@echo "Status:     $(shell docker inspect $(SIMULATOR_CONTAINER) \
	  --format "{{.State.Status}}" 2>/dev/null || echo "not found")"

# Soft restart — creates fresh container, re-runs provisioning
# Use after a rebuild or to pick up simulation config value changes
simulator-restart:
	$(MAKE) simulator-start SIM=$(or $(CURRENT_SIM),default)

# Hard restart — teardown first, then fresh start — guaranteed clean slate
# Use when device capabilities or room topology have changed
simulator-restart-hard:
	$(MAKE) simulator-teardown SIM=$(or $(CURRENT_SIM),default)
	$(MAKE) simulator-start SIM=$(or $(CURRENT_SIM),default)

# Soft switch — stop current, start new simulation, old data stays in DB
# Usage: make simulator-switch SIM=load-test
simulator-switch:
	$(MAKE) simulator-stop
	$(MAKE) simulator-start SIM=$(SIM)

# Hard switch — teardown current, start new simulation, clean slate
# Usage: make simulator-switch-hard SIM=load-test
simulator-switch-hard:
	$(MAKE) simulator-teardown SIM=$(or $(CURRENT_SIM),default)
	$(MAKE) simulator-start SIM=$(SIM)

# Teardown simulation data — deletes all provisioned users via API cascade
# Stops simulator first if running. Uses current simulation if SIM not specified.
# Usage: make simulator-teardown
# Usage: make simulator-teardown SIM=load-test
simulator-teardown:
	$(COMPOSE_ALL) stop simulator-service 2>/dev/null || true
	$(COMPOSE_ALL) run --rm simulator-service ./bin/simulator-service \
	  --mode=teardown --simulation=$(or $(SIM),$(CURRENT_SIM),default)

# ── Docker utilities ──────────────────────────────────────────────────────────

# List running containers
docker-ps:
	docker ps

# List all containers including stopped
docker-ps-all:
	docker ps -a

# Live resource usage per container
docker-stats:
	docker stats

# List images
docker-images:
	docker images

# List all images including intermediates
docker-images-all:
	docker images -a

# Remove stopped containers, dangling images, unused networks
docker-prune:
	docker system prune

# List volumes
docker-volumes:
	docker volume ls

# Remove unused volumes
docker-volume-prune:
	docker volume prune

# ── Mosquitto ────────────────────────────────────────────────────────────────

# Generate mosquitto password file from .env credentials
# Run this whenever MQTT passwords are changed in .env
mosquitto-passwd:
	docker run --rm eclipse-mosquitto:2 sh -c " \
	  mosquitto_passwd -c -b /tmp/passwd $${MQTT_DEVICE_USERNAME} $${MQTT_DEVICE_PASSWORD} && \
	  mosquitto_passwd -b /tmp/passwd $${MQTT_DEVICE_SERVICE_USERNAME} $${MQTT_DEVICE_SERVICE_PASSWORD} && \
	  mosquitto_passwd -b /tmp/passwd healthcheck healthcheck && \
	  cat /tmp/passwd" | grep -v "^Adding" > deployments/mosquitto/passwd

# ── MQTT subscriptions ────────────────────────────────────────────────────────

# Subscribe to all telemetry from all devices
mqtt-telemetry:
	docker exec -it climate-control-mosquitto-1 mosquitto_sub \
	  -h localhost -t 'devices/+/telemetry' \
	  -u $${MQTT_DEVICE_SERVICE_USERNAME} -P $${MQTT_DEVICE_SERVICE_PASSWORD} \
	  -v

# Subscribe to all commands to all devices
mqtt-commands:
	docker exec -it climate-control-mosquitto-1 mosquitto_sub \
	  -h localhost -t 'devices/+/cmd' \
	  -u $${MQTT_DEVICE_USERNAME} -P $${MQTT_DEVICE_PASSWORD} \
	  -v

# Subscribe to all topics
mqtt-all:
	docker exec -it climate-control-mosquitto-1 mosquitto_sub \
	  -h localhost -t 'devices/#' \
	  -u $${MQTT_DEVICE_SERVICE_USERNAME} -P $${MQTT_DEVICE_SERVICE_PASSWORD} \
	  -v

# Subscribe to a specific device — Usage: make mqtt-device HW_ID=sim-default-0-0-0
mqtt-device:
	docker exec -it climate-control-mosquitto-1 mosquitto_sub \
	  -h localhost -t 'devices/$(HW_ID)/#' \
	  -u $${MQTT_DEVICE_SERVICE_USERNAME} -P $${MQTT_DEVICE_SERVICE_PASSWORD} \
	  -v

