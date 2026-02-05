FROM golang:1.24-alpine AS builder
WORKDIR /app

# Copiamos sólo los módulos necesarios para resolver dependencias
COPY execution-workers/go.mod ./go.mod
COPY platform /platform
RUN go mod download

# Copiamos el código del servicio
COPY execution-workers/. .
RUN go build -o /bin/execution-workers ./cmd/worker

FROM alpine:3.19
WORKDIR /app
ENV GITHUB_TOKEN=""
COPY --from=builder /bin/execution-workers /app/execution-workers
EXPOSE 8082
ENTRYPOINT ["/app/execution-workers"]
