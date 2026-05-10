# syntax=docker/dockerfile:1

FROM golang:1.24.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /app/E-Nav .

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/E-Nav ./E-Nav
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/images ./images

RUN mkdir -p /app/data

VOLUME ["/app/data"]

EXPOSE 1239

CMD ["./E-Nav"]
