# syntax=docker/dockerfile:1

FROM golang:1.24-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
	-ldflags="-w -s" \
	-o server \
	./cmd/api

FROM debian:bookworm-slim AS production

RUN apt-get update && apt-get install -y --no-install-recommends \
	ca-certificates \
	poppler-utils \
	tesseract-ocr \
	tesseract-ocr-ind \
	tesseract-ocr-eng \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/server ./server

ENV PORT=8080
EXPOSE 8080

USER nobody

CMD ["./server"]
