# ----------------------
# Build stage: compile Go application
# ----------------------
FROM golang:1.24 AS builder

WORKDIR /app

# 安装构建依赖 (libopus 开发包 + pkg-config)
RUN set -eux; \
    for attempt in 1 2 3; do \
      apt-get update && apt-get install -y --no-install-recommends \
        pkg-config \
        libopus-dev \
      && break; \
      if [ "${attempt}" -eq 3 ]; then exit 1; fi; \
      rm -rf /var/lib/apt/lists/*; \
      sleep 5; \
    done; \
    rm -rf /var/lib/apt/lists/*

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

# Build the Go application
RUN go build -ldflags="-s -w" -o TinyClaw ./cmd/tinyclaw
RUN go build -ldflags="-s -w" -o TinyClawNode ./cmd/tinyclaw-node
RUN go build -ldflags="-s -w" -o TinyClawAdmin ./admin

FROM node:20-bookworm-slim AS node_runtime

# ----------------------
# Runtime stage
# ----------------------
FROM builder AS runtime

ARG MCP_FILESYSTEM_VERSION=2026.1.14
ARG MCP_MEMORY_VERSION=2026.1.26
ARG AMAP_MCP_VERSION=0.0.8
ARG UV_VERSION=0.11.3
ARG MCP_FETCH_VERSION=2025.4.7
ARG MCP_TIME_VERSION=2026.1.26
ARG ARXIV_MCP_VERSION=0.4.11
ARG PYTHON_MCP_VERSION=1.27.0
ARG GITHUB_MCP_SERVER_VERSION=v0.33.0

WORKDIR /app

COPY --from=node_runtime /usr/local /usr/local

RUN set -eux; \
    export DEBIAN_FRONTEND=noninteractive; \
    for attempt in 1 2 3; do \
      apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        git \
        python-is-python3 \
        python3-pip \
      && break; \
      if [ "${attempt}" -eq 3 ]; then exit 1; fi; \
      rm -rf /var/lib/apt/lists/*; \
      sleep 5; \
    done; \
    for attempt in 1 2 3; do \
      python3 -m pip install --break-system-packages --retries 10 --timeout 120 \
        mcp==${PYTHON_MCP_VERSION} \
        uv==${UV_VERSION} \
        mcp-server-fetch==${MCP_FETCH_VERSION} \
        mcp-server-time==${MCP_TIME_VERSION} \
      && break; \
      if [ "${attempt}" -eq 3 ]; then exit 1; fi; \
      sleep 5; \
    done; \
    for attempt in 1 2 3; do \
      python3 -m pip install --break-system-packages --retries 10 --timeout 120 \
        sgmllib3k==1.0.0 \
        arxiv-mcp-server==${ARXIV_MCP_VERSION} \
      && break; \
      if [ "${attempt}" -eq 3 ]; then exit 1; fi; \
      sleep 5; \
    done; \
    for attempt in 1 2 3; do \
      npm install -g \
        @modelcontextprotocol/server-filesystem@${MCP_FILESYSTEM_VERSION} \
        @modelcontextprotocol/server-memory@${MCP_MEMORY_VERSION} \
        @amap/amap-maps-mcp-server@${AMAP_MCP_VERSION} \
      && break; \
      if [ "${attempt}" -eq 3 ]; then exit 1; fi; \
      sleep 5; \
    done; \
    for attempt in 1 2 3; do \
      GOBIN=/usr/local/bin go install github.com/github/github-mcp-server/cmd/github-mcp-server@${GITHUB_MCP_SERVER_VERSION} \
      && break; \
      if [ "${attempt}" -eq 3 ]; then exit 1; fi; \
      sleep 5; \
    done; \
    mkdir -p /app/data/mcp/memory /app/data/mcp/arxiv /app/data/sessions /workspace; \
    rm -rf /var/lib/apt/lists/* /root/.cache/pip /root/.npm

COPY deploy/docker/start-tinyclaw.sh /app/start-tinyclaw.sh
COPY deploy/docker/start-services.sh /app/start-services.sh

RUN mkdir -p /app/data /app/data/sessions /app/log && \
    chmod +x /app/start-tinyclaw.sh /app/start-services.sh

# Expose application port
EXPOSE 36060
EXPOSE 18080

# Set entrypoint
ENTRYPOINT ["/app/start-services.sh"]
