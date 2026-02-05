FROM golang:1.24-alpine AS builder
WORKDIR /app

# Copiamos sólo los módulos necesarios para resolver dependencias
COPY control-plane-api/go.mod ./go.mod
COPY platform /platform
RUN go mod download

# Copiamos el código del servicio
COPY control-plane-api/. .
RUN go build -o /bin/control-plane-api ./cmd/api

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /bin/control-plane-api /app/control-plane-api
EXPOSE 8080
ENTRYPOINT ["/app/control-plane-api"]
