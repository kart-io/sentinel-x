FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY staging/ staging/
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/sentinel-api ./cmd/api

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/bin/sentinel-api /app/bin/sentinel-api
COPY configs/ /app/configs/

EXPOSE 8081
EXPOSE 9091

CMD ["/app/bin/sentinel-api", "-c", "/app/configs/sentinel-api.yaml"]
