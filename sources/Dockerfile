FROM golang:1.23 AS build

ENV GOTOOLCHAIN=auto

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations

RUN CGO_ENABLED=0 go build -o /bin/sources ./cmd/api

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates wget && rm -rf /var/lib/apt/lists/*

COPY --from=build /bin/sources /usr/local/bin/sources

EXPOSE 8082

CMD ["/usr/local/bin/sources"]