# ----------------------
# Build stage: compile Go application
# ----------------------
FROM golang:1.24 AS builder

WORKDIR /app

# 安装构建依赖 (libopus 开发包 + pkg-config)
RUN apt-get update && apt-get install -y --no-install-recommends \
    pkg-config \
    libopus-dev \
 && rm -rf /var/lib/apt/lists/*

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

# Build the Go application
RUN go build -ldflags="-s -w" -o TinyClaw ./cmd/tinyclaw
RUN go build -ldflags="-s -w" -o TinyClawAdmin ./admin

# ----------------------
# Runtime stage
# ----------------------
FROM builder AS runtime

WORKDIR /app

COPY deploy/docker/start-tinyclaw.sh /app/start-tinyclaw.sh
COPY deploy/docker/start-services.sh /app/start-services.sh

RUN mkdir -p /app/data /app/log && \
    chmod +x /app/start-tinyclaw.sh /app/start-services.sh

# Expose application port
EXPOSE 36060
EXPOSE 18080

# Set entrypoint
ENTRYPOINT ["/app/start-services.sh"]
