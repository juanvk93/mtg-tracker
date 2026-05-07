# === Dockerfile multistage para MTG Tracker ===
# pgx es Go puro → no necesita CGO ni gcc → imagen final mucho más ligera

# Stage 1: Build
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copiar dependencias primero (mejor caché de Docker)
# IMPORTANTE: go.sum debe estar commiteado en el repositorio (ejecuta go mod tidy localmente antes del push)
COPY go.mod go.sum ./
RUN go mod download

# Copiar el resto del código fuente
COPY . .

# Compilar binario estático (CGO deshabilitado, pgx no lo necesita)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/mtg-tracker ./cmd/server

# === Stage 2: Imagen final mínima ===
FROM alpine:3.20

WORKDIR /app

# Certificados TLS para conectar con Supabase vía SSL
RUN apk add --no-cache ca-certificates tzdata

# Crear usuario no-root por seguridad
RUN adduser -D -s /bin/sh appuser

# Copiar binario compilado
COPY --from=builder /app/mtg-tracker .

# Copiar assets estáticos y templates
COPY --chown=appuser:appuser static/ ./static/
COPY --chown=appuser:appuser templates/ ./templates/

USER appuser

# Variables de entorno con valores por defecto
ENV PORT=8080

EXPOSE 8080

CMD ["/app/mtg-tracker"]
