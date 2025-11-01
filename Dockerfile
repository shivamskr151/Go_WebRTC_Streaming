# Multi-stage Docker build for Go WebRTC Streaming Server

# Build React frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

# Copy package files
COPY web/package.json ./
COPY web/package-lock.json* ./

# Install dependencies
RUN npm install

# Copy frontend source (exclude node_modules)
COPY web/src ./src
COPY web/index.html ./
COPY web/tailwind.config.js ./
COPY web/postcss.config.js ./
COPY web/tsconfig.json ./
COPY web/tsconfig.node.json ./
COPY web/vite.config.ts ./

# Build React app
RUN npm run build

# Build Go backend
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy built React frontend from frontend-builder
COPY --from=frontend-builder /app/web/dist ./web/dist

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o webrtc-server cmd/server/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata ffmpeg

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/webrtc-server .

# Copy web assets
COPY --from=builder /app/web ./web

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8080 1935

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/status || exit 1

# Run the application
CMD ["./webrtc-server"]
