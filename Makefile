# Variaveis
DASHBOARD_DIR := dashboard
GO_MAIN := ./cmd/FreddyBot/main.go
BINARY := Release
IMAGE := freddybot:local
PORT := 7000
GIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS := -ldflags "-X github.com/leirbagxis/FreddyBot/internal/utils.Version=$(GIT_HASH)"

.PHONY: all build build-ui build-server run dev clean docker-build docker-run help

all: build

build: build-ui build-server
	@echo "✅ Build completo concluido ($(GIT_HASH))"

$(DASHBOARD_DIR)/node_modules:
	@echo "📦 Instalando dependencias do Dashboard..."
	cd $(DASHBOARD_DIR) && npm install

build-ui: $(DASHBOARD_DIR)/node_modules
	@echo "🚀 Iniciando build do Dashboard..."
	cd $(DASHBOARD_DIR) && npm run build
	@echo "✅ Build do Dashboard concluido!"

build-server:
	@echo "📦 Construindo o binario do FreddyBot..."
	go build $(LDFLAGS) -o $(BINARY) $(GO_MAIN)
	@echo "✅ Build do servidor concluido!"

run: build
	@echo "🤖 Iniciando o FreddyBot ($(GIT_HASH))..."
	./$(BINARY)

dev: build-ui
	@echo "🛠️ Iniciando o FreddyBot em modo desenvolvimento..."
	go run $(LDFLAGS) $(GO_MAIN)

docker-build:
	@echo "🐳 Construindo imagem Docker $(IMAGE)..."
	docker build --build-arg GIT_HASH=$(GIT_HASH) -t $(IMAGE) .

docker-run:
	@echo "🐳 Iniciando container Docker $(IMAGE)..."
	docker run --rm -p $(PORT):$(PORT) --env-file .env $(IMAGE)

clean:
	@echo "🧹 Limpando arquivos gerados..."
	rm -rf $(DASHBOARD_DIR)/dist
	rm -f $(BINARY)

help:
	@echo "Uso: make [comando]"
	@echo ""
	@echo "Comandos:"
	@echo "  all          - Build total (padrao)"
	@echo "  build        - Build total (UI + servidor)"
	@echo "  run          - Build total e executa o bot"
	@echo "  dev          - Build da UI e executa via go run"
	@echo "  build-ui     - Apenas faz o build do dashboard"
	@echo "  build-server - Apenas constroi o binario do bot"
	@echo "  docker-build - Constroi a imagem Docker local"
	@echo "  docker-run   - Executa a imagem Docker local"
	@echo "  clean        - Remove arquivos gerados"
