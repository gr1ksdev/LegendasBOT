# 🤖 FreddyBot - V2 Motor

> **CONFIDENTIAL & PROPRIETARY**
> This repository contains proprietary code. Unauthorized access, distribution, or copying is strictly prohibited.

FreddyBot é um motor avançado de automação para Telegram, focado em gerenciamento de canais, processamento de legendas e construção de postagens interativas (Post Builder).

---

## 🚀 Funcionalidades Principais

- **V2 Pipeline Engine**: Processamento modular de posts (Sticker Separator, Dynamic Links, Global Captions).
- **Post Builder**: Interface interativa para criação de conteúdos com mídias e botões.
- **Dual-Layer Cache**: Implementação de cache em L1 (RAM) e L2 (Redis) para alta performance.
- **Multi-Database**: Suporte nativo a SQLite (desenvolvimento) e PostgreSQL (produção).
- **Admin Dashboard**: Painel centralizado para controle global de configurações e usuários.

---

## 🛠️ Instalação e Deploy

### 1. Configuração do Ambiente
Clone o repositório e configure as variáveis de ambiente necessárias:

```bash
cp .env-example .env
```

Edite o arquivo `.env` com as seguintes chaves obrigatórias:
- `TELEGRAM_BOT_TOKEN`: Token gerado pelo @BotFather.
- `REDIS_HOST`: Conexão com a instância Redis.
- `APP_ENV`: `prod` para uso de PostgreSQL.
- `DATABASE_FILE`: String de conexão DSN do PostgreSQL (ex: `host=... user=... password=... dbname=...`).

### 2. Execução via Docker (Testes/Local)
Para ambientes que suportam Docker, utilize o compose para subir as dependências rapidamente:
```bash
docker-compose up -d
```

### 3. Build e Execução Manual
```bash
go mod tidy
go build -o freddybot ./cmd/FreddyBot/main.go
./freddybot
```

---

## 📂 Estrutura de Diretórios

- `cmd/FreddyBot`: Ponto de entrada da aplicação Go.
- `internal/api`: Servidor REST e lógica do Admin Dashboard.
- `internal/core`: Serviços de domínio e lógica de negócio.
- `internal/telegram`: Handlers, middlewares e motor de eventos do bot.
- `pkg/`: Pacotes utilitários de configuração e logging.
- `dashboard/`: Código-fonte do Dashboard (React/Vite).

---

## 📜 Licença

© 2026 FreddyBot Development Team. Todos os direitos reservados.
Código de uso estritamente privado e confidencial.
