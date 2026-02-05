FROM golang:1.24-alpine AS builder
WORKDIR /app

# Copiamos sólo los módulos necesarios para resolver dependencias
COPY workflow-engine/go.mod ./go.mod
COPY platform /platform
RUN go mod download

# Copiamos el código del servicio
COPY workflow-engine/. .
RUN go build -o /bin/workflow-engine ./cmd/engine

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /bin/workflow-engine /app/workflow-engine
EXPOSE 8081
ENTRYPOINT ["/app/workflow-engine"]
