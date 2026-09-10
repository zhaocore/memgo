MODULE := github.com/zhao-core/memgo
BIN_DIR := bin
CONTRACT_STACK := tests/contract/compose.contract.yaml

.PHONY: build test vet lint contract contract-up contract-capture contract-down clean

build:
	go build -o $(BIN_DIR)/memgo-server ./cmd/server
	go build -o $(BIN_DIR)/memgo ./cli/go/cmd/memgo

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run

# 契约栈重置 (全新库) + 拉起; 套件设计为一次性 (register/jti 等), 每轮必须新库
contract-up:
	podman compose -f $(CONTRACT_STACK) down -v || true
	podman compose -f $(CONTRACT_STACK) up -d --build

contract-capture: contract-up
	./tests/contract/run.sh --capture

# 黑盒契约套件: 默认打 Python server (回归), CONTRACT_IMPL=go 打 Go server
contract: contract-up
	./tests/contract/run.sh $(if $(CONTRACT_IMPL),--impl $(CONTRACT_IMPL),)

contract-down:
	podman compose -f $(CONTRACT_STACK) down -v

# ---------- P4: 生产/切换 ----------

DEPLOY_STACK := deploy/docker-compose.yaml
# 宿主 API 端口 (8888 被占的机器用 API_PORT=18888 make up)
API_PORT ?= 8888

# 一键起栈 (memgo-server + pgvector; dashboard 用 --profile dashboard)
up:
	cd deploy && API_HOST_PORT=$(API_PORT) podman compose -f docker-compose.yaml up -d --build
	@make -s wait-api

up-dashboard:
	cd deploy && podman compose -f docker-compose.yaml --profile dashboard up -d --build

# 对照开关: 上游 python server 起同一套栈
up-py:
	podman compose -f tests/contract/compose.contract.yaml up -d --build

down:
	cd deploy && podman compose -f docker-compose.yaml down -v

# 单容器生产路径等价 (连外部 Postgres): 裸 docker run
run-local:
	CGO_ENABLED=0 go build -o $(BIN_DIR)/memgo-server ./cmd/server
	@echo "运行: POSTGRES_HOST=<host> POSTGRES_PASSWORD=<pw> JWT_SECRET=<secret> OPENAI_API_KEY=<key> $(BIN_DIR)/memgo-server"

# seed: setup-status → register → 建 key
bootstrap: up
	API=http://localhost:$(API_PORT) ./deploy/seed.sh

wait-api:
	@for i in $$(seq 1 60); do curl -sf -m 2 http://localhost:$(API_PORT)/auth/setup-status >/dev/null 2>&1 && exit 0; sleep 1; done; echo "memgo-server 未就绪"; exit 1

# 三探: API / postgres / dashboard (dashboard 未起时该项失败)
health:
	@curl -sf http://localhost:$(API_PORT)/auth/setup-status | grep -q needsSetup && echo "api: ok"
	@podman exec $$(podman ps --format '{{.Names}}' | grep memgo-postgres) pg_isready -q && echo "postgres: ok"
	@curl -sf http://localhost:3000/api/health >/dev/null 2>&1 && echo "dashboard: ok" || echo "dashboard: 未运行 (make up-dashboard)"

# ---------- 三 CLI (python / node 迁入 + go) ----------

# python CLI 测试 (venv 隔离, 不污染用户环境)
cli-py-test:
	python3 -m venv /tmp/memgopycli && /tmp/memgopycli/bin/pip install -q -e cli/python pytest
	cd cli/python && /tmp/memgopycli/bin/python -m pytest tests -q

# node CLI 测试 (vitest)
cli-node-test:
	cd cli/node && pnpm install --silent && npx vitest run

# go CLI parity + 单测
cli-go-test:
	go test ./cli/go/ -v

# 存量 python (alembic) 部署 → Go (goose) 版本表迁移
migrate:
	go build -o $(BIN_DIR)/memgo-migrate ./cmd/migrate
	APP_DB_DSN="$${APP_DB_DSN}" $(BIN_DIR)/memgo-migrate

clean:
	rm -rf $(BIN_DIR)
