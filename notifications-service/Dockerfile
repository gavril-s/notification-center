FROM golang:1.22-alpine AS build

WORKDIR /app

RUN apk add --no-cache git

ENV GOPROXY=direct
ENV GOTOOLCHAIN=local

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /notifications-service ./cmd/api

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=build /notifications-service .

EXPOSE 8083

CMD ["./notifications-service"]