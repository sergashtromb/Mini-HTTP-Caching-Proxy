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

RUN mkdir -p /app/tmp && chown -R appuser:appuser /app

ENV port=8080 \
	LOG_LEVEL=error \
	CACHE_IN_RAM=0 \
	TMP_PATH=/tmp \
	LIST_HOSTS=example.com

ENTRYPOINT ["/app/proxy"]

