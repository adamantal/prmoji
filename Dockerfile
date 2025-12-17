FROM golang:1.25.5-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/prmoji ./cmd/prmoji


FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN useradd --create-home --home-dir /app --shell /usr/sbin/nologin --uid 10001 prmoji

WORKDIR /app
COPY --from=builder /out/prmoji /app/prmoji

EXPOSE 5000

USER prmoji

ENTRYPOINT ["/app/prmoji"]
