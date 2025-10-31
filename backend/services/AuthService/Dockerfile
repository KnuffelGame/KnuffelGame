FROM golang:1.25.3-alpine AS builder

COPY . /app
WORKDIR /app/services/AuthService
RUN go build -o authservice ./cmd/AuthService

FROM alpine:3.22.2

WORKDIR /app
RUN apk --no-cache add curl
COPY --from=builder /app/services/AuthService/authservice .

HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 CMD curl -f http://localhost:8081/healthcheck || exit 1

EXPOSE 8081

CMD ["./authservice"]