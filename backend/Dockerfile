# ==============================================================================
# CogniShell Go Backend - Dockerfile
# ==============================================================================

# Stage 1: Builder
FROM golang:1.22-bookworm AS builder

WORKDIR /app

# Copy dependency files and download
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Compile Go application
RUN CGO_ENABLED=0 GOOS=linux go build -o cognishell .

# Stage 2: Runner
# Use Microsoft's official PowerShell base image so pwsh is pre-installed
FROM mcr.microsoft.com/powershell:7.4-debian-11

WORKDIR /app

# Copy compiled binary from builder stage
COPY --from=builder /app/cognishell .

# Expose API port
EXPOSE 8080

# Run the backend
ENTRYPOINT ["./cognishell"]
