# syntax=docker/dockerfile:1.6
ARG GO_VERSION=1.22
FROM golang:${GO_VERSION}-alpine AS builder
RUN apk add --no-cache build-base git
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG SERVICE_PATH=api-gateway
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/service ./cmd/${SERVICE_PATH}

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /out/service /app/service
ENV SERVER_HOST=0.0.0.0
EXPOSE 8080
ENTRYPOINT ["/app/service"]
