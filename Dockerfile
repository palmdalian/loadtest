# Use the latest official Golang image for building Go applications
FROM golang:latest AS go_builder

# Set the working directory
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the application source code
COPY . .

# Build the Go application for the target architecture
ARG TARGETARCH
RUN GOARCH=${TARGETARCH} go build -o /app/loadtester cmd/main.go

# Use an Ubuntu image for building apib with cmake
FROM ubuntu:20.04 AS apib_builder

# Non-interactive mode for apt-get
ENV DEBIAN_FRONTEND=noninteractive

# Install dependencies for cmake build
RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    libev-dev \
    libssl-dev \
    git \
    ninja-build \
    && rm -rf /var/lib/apt/lists/*

# Clone and build apib using cmake
WORKDIR /build
RUN git clone https://github.com/apigee/apib.git
WORKDIR /build/apib
RUN mkdir release && cd release && \
    cmake .. -DCMAKE_BUILD_TYPE=Release -G Ninja && \
    ninja apib

# Use a lightweight Debian image for the runtime
FROM debian:bullseye-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    libev4 \
    libssl1.1 \
    && rm -rf /var/lib/apt/lists/*

# Copy apib binary directly to /usr/local/bin
COPY --from=apib_builder /build/apib/release/apib /usr/local/bin/

# Copy the Go application binary
COPY --from=go_builder /app/loadtester /usr/local/bin/

# Ensure /usr/local/bin is in the PATH
ENV PATH="/usr/local/bin:$PATH"