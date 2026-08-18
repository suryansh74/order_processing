FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /order-api ./api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /order-api /order-api
EXPOSE 8000
ENTRYPOINT ["/order-api"]
