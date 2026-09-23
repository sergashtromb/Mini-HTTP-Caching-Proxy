FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/proxy .

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /out/proxy /app

RUN adduser -D -u 1000 appuser

RUN mkdir -p /app/tmp && chown -R appuser:appuser /app

ENV port=8080 \
	LOG_LEVEL=error \
	CACHE_IN_RAM=0 \
	TMP_PATH=/tmp \
	LIST_HOSTS=example.com

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["/app/proxy"]

