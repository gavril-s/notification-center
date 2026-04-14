FROM golang:1.22-alpine AS build

ENV GOTOOLCHAIN=auto

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations

RUN CGO_ENABLED=0 go build -o /bin/sources ./cmd/api

FROM alpine:3.20

RUN apk --no-cache add ca-certificates

COPY --from=build /bin/sources /usr/local/bin/sources

EXPOSE 8082

CMD ["/usr/local/bin/sources"]