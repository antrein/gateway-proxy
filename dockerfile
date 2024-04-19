# Use an official Golang runtime as a parent image
FROM golang:1.21-alpine as builder

# Set the working directory
WORKDIR /app

# Copy the source code into the container
COPY . .

# Download dependencies
RUN go mod tidy

# Build the application
RUN go build -o gateway-proxy

# Use a smaller image to run the compiled application
FROM alpine:latest  

WORKDIR /root/

COPY --from=builder /app/gateway-proxy .

EXPOSE 8080

CMD ["./gateway-proxy"]
