# Stage 1: Build React 19 Frontend
FROM node:22-alpine AS web-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go Binaries
FROM golang:1.24-alpine AS go-builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY internal/ internal/
COPY cmd/ cmd/
COPY --from=web-builder /app/internal/ui/dist ./internal/ui/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/auditforge ./cmd/auditforge
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/auditverify ./cmd/auditverify

# Stage 3: Minimal Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /bin/auditforge /usr/local/bin/auditforge
COPY --from=go-builder /bin/auditverify /usr/local/bin/auditverify
COPY config.yaml /app/config.yaml

EXPOSE 8080 9090
VOLUME ["/app/data"]

ENTRYPOINT ["/usr/local/bin/auditforge"]
CMD ["serve", "--config", "/app/config.yaml"]
