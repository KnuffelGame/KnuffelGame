FROM golang:1.25.3-alpine AS builder

COPY . /app
WORKDIR /app/services/SSEService
RUN go build -o sseservice ./cmd/SSEService

FROM alpine:3.22.2

WORKDIR /app
RUN apk --no-cache add ca-certificates curl

# Create non-root user
RUN addgroup -g 1000 appgroup && adduser -u 1000 -G appgroup -s /bin/sh -D appuser

COPY --from=builder /app/services/SSEService/sseservice .

HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8084/healthcheck || exit 1

USER appuser
EXPOSE 8084

CMD ["./sseservice"]