FROM golang:1.22-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
COPY internal internal/
COPY cmd cmd/

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/recipients ./cmd/api

FROM alpine:3.20

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=build /bin/recipients /usr/local/bin/recipients

EXPOSE 8081

CMD ["/usr/local/bin/recipients"]