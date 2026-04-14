FROM golang:1.23 AS build

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends git && rm -rf /var/lib/apt/lists/*

ENV GOPROXY=https://proxy.golang.org,direct

COPY go.mod go.sum ./
RUN GOSUMDB=off go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /notifications-service ./cmd/api

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata wget && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /notifications-service .

EXPOSE 8083

CMD ["./notifications-service"]