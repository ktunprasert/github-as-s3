# Stage 1: Build the application
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

RUN go build -o ./tmp/main ./cmd/cli/

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/tmp/main /app/main

# The application reads configuration from environment variables.
# Refer to your readme.md or .env.example for required variables like:
# GITHUB_TOKEN, GITHUB_OWNER, GHS3_PORT, GHS3_ADDRESS
# These should be provided when running the container, e.g., using 'docker run -e VAR=value'.

# If your application listens on a specific port (e.g., GHS3_PORT),
# you might want to EXPOSE it. For example, if GHS3_PORT defaults to 8080:
# EXPOSE 8080

ENTRYPOINT ["/app/main"]
