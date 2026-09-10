MODULE := github.com/zhao-core/memgo
BIN_DIR := bin
CONTRACT_STACK := tests/contract/compose.contract.yaml

.PHONY: build test vet lint contract contract-up contract-capture contract-down clean

build:
	go build -o $(BIN_DIR)/memgo-server ./cmd/server
	go build -o $(BIN_DIR)/memgo ./cmd/memgo

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

clean:
	rm -rf $(BIN_DIR)
