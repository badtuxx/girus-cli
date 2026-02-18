FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o girus -ldflags="-X 'github.com/badtuxx/girus-cli/internal/common.Version=${VERSION}'" ./main.go

# Use a minimal alpine image for the final container
FROM alpine:3.22

# Install necessary packages
RUN apk add --no-cache ca-certificates curl bash docker-cli

# Install kind and kubectl
RUN curl -Lo /usr/local/bin/kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64 && \
    chmod +x /usr/local/bin/kind && \
    curl -Lo /usr/local/bin/kubectl "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod +x /usr/local/bin/kubectl

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/girus /usr/local/bin/girus

# Create entrypoint
ENTRYPOINT ["/usr/local/bin/girus"]

# Default command
CMD ["help"]
