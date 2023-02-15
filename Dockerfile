FROM golang:1.19-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the source code into the container
COPY . .

# Build the binary
RUN go build -o droid

# Final image
FROM alpine:3.17

# Copy over binaries from the builder
COPY --from=builder /app/droid /usr/bin/droid

EXPOSE 8080
ENTRYPOINT ["droid"]

