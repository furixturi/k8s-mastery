# Use the full golang base image as the build environment 
FROM golang:1.16 AS builder

# Create and set the current working directory
WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN export GOPROXY=direct && \
    go mod download

# Copy application code 
COPY *.go ./

# Build the application and put it at the root for a shorter cmd to run it
# RUN go build -o sa-webapp-go
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' .

# Then use alpine as the runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates bash

WORKDIR /root/

# copy the pre-build binary from the previous stage
COPY --from=builder /app/sa-webapp-go .

# Expose the port that the app will be listening on
# This is just for readability
EXPOSE 8080

# Start the app
CMD [ "./sa-webapp-go" ]