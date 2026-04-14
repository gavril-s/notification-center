FROM golang:1.23 AS build

WORKDIR /app

COPY go.mod go.sum ./
COPY internal internal/
COPY cmd cmd/

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/recipients ./cmd/api

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates wget && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /bin/recipients /usr/local/bin/recipients

EXPOSE 8081

CMD ["/usr/local/bin/recipients"]