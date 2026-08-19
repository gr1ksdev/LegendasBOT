# Estagio 1: build da Dashboard (Frontend)
FROM node:24-alpine3.23 AS frontend-builder
WORKDIR /app/dashboard

COPY dashboard/package*.json ./
RUN npm ci

COPY dashboard/ ./
RUN npm run build

# Estagio 2: build do servidor (Go)
FROM golang:1.25.7-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /app/dashboard/dist ./dashboard/dist

ARG GIT_HASH=unknown
RUN go build -ldflags "-X github.com/leirbagxis/FreddyBot/internal/utils.Version=${GIT_HASH}" -o Release ./cmd/FreddyBot/main.go

# Estagio 3: imagem final
FROM alpine:3.23
WORKDIR /app

RUN apk add --no-cache ca-certificates && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /app/Release ./Release
COPY --from=builder /app/dashboard/dist ./dashboard/dist
COPY --from=builder /app/config ./config

USER appuser

EXPOSE 7000

CMD ["./Release"]
