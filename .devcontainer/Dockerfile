FROM golang:1.25-trixie AS golang
FROM debian:trixie

COPY --from=golang /usr/local/go /usr/local/go

WORKDIR /app
ARG TARGETPLATFORM
ENV DEBIAN_FRONTEND="noninteractive"
ENV PATH="$PATH:/usr/local/go/bin"
ENV GOBIN="/usr/local/go/bin"

RUN \
    # Check if architecture is supported
    set -e && \
    echo "Building for ${TARGETPLATFORM}" && \
    if [ "${TARGETPLATFORM}" != "linux/amd64" ] && [ "${TARGETPLATFORM}" != "linux/arm64" ]; then \
      echo "Unsupported architecture: ${TARGETPLATFORM}" && \
      exit 1; \
    fi && \
    # Create temporary directory for downloads
    mkdir -p /tmp/downloads && \
    cd /tmp/downloads && \
    # Install system dependencies
    apt update -qq && \
    apt install -qq -y wget curl net-tools tree zip unzip p7zip-full git python3 python3-pip && \
    # Install mkdocs-material
    pip install -qq mkdocs-material --break-system-packages && \
    pip install -qq mkdocs-llmstxt --break-system-packages && \
    # Download and install architecture-specific binaries
    if [ "${TARGETPLATFORM}" = "linux/arm64" ]; then \
      echo "Downloading arm64 binaries" && \
      # Install task
      wget --no-verbose https://github.com/go-task/task/releases/download/v3.45.5/task_linux_arm64.tar.gz && \
      tar -xzf task_linux_arm64.tar.gz && \
      mv ./task /usr/local/bin/task && \
      # Install golangci-lint
      wget --no-verbose https://github.com/golangci/golangci-lint/releases/download/v2.6.2/golangci-lint-2.6.2-linux-arm64.tar.gz && \
      tar -xzf golangci-lint-2.6.2-linux-arm64.tar.gz && \
      mv ./golangci-lint-2.6.2-linux-arm64/golangci-lint /usr/local/bin/golangci-lint && \
      # Install dprint code formatter
      wget --no-verbose https://github.com/dprint/dprint/releases/download/0.50.2/dprint-aarch64-unknown-linux-gnu.zip && \
      unzip -q dprint-aarch64-unknown-linux-gnu.zip && \
      mv ./dprint /usr/local/bin/dprint; \
    else \
      echo "Downloading amd64 binaries" && \
      # Install task
      wget --no-verbose https://github.com/go-task/task/releases/download/v3.45.5/task_linux_amd64.tar.gz && \
      tar -xzf task_linux_amd64.tar.gz && \
      mv ./task /usr/local/bin/task && \
      # Install golangci-lint
      wget --no-verbose https://github.com/golangci/golangci-lint/releases/download/v2.6.2/golangci-lint-2.6.2-linux-amd64.tar.gz && \
      tar -xzf golangci-lint-2.6.2-linux-amd64.tar.gz && \
      mv ./golangci-lint-2.6.2-linux-amd64/golangci-lint /usr/local/bin/golangci-lint && \
      # Install dprint code formatter
      wget --no-verbose https://github.com/dprint/dprint/releases/download/0.50.2/dprint-x86_64-unknown-linux-gnu.zip && \
      unzip -q dprint-x86_64-unknown-linux-gnu.zip && \
      mv ./dprint /usr/local/bin/dprint; \
    fi && \
    # Make binaries executable
    chmod +x /usr/local/bin/* && \
    # Set default git config (https://github.com/golangci/golangci-lint/issues/4033)
    git config --global --add safe.directory '*' && \
    # Clean up downloads and apt cache
    cd /app && \
    rm -rf /tmp/downloads && \
    rm -rf /var/lib/apt/lists/*

##############
# START HERE #
##############

# Add bashrc and keep container running
COPY .devcontainer/.bashrc /root/.bashrc
CMD ["sleep", "infinity"]
