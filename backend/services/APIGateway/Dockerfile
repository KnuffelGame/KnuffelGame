FROM golang:1.25.3-alpine AS builder

COPY . /app
WORKDIR /app/services/APIGateway
RUN go build -o apigateway ./cmd/APIGateway

FROM alpine:3.22.2

WORKDIR /app
RUN apk --no-cache add ca-certificates curl

# Create non-root user
RUN addgroup -g 1000 appgroup && adduser -u 1000 -G appgroup -s /bin/sh -D appuser

COPY --from=builder /app/services/APIGateway/apigateway .

HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthcheck || exit 1

USER appuser
EXPOSE 8080

CMD ["./apigateway"]