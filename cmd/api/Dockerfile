# Use base golang image from Docker Hub
FROM golang:1.15 as build

WORKDIR /app

# Copy the go.mod and go.sum, download the dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy rest of the application source code
COPY . ./

# Compile the application to /app/bin.
RUN go build -mod=readonly -v -o /app/bin ./cmd/api

# Now create separate deployment image
FROM gcr.io/distroless/base

WORKDIR /app
COPY --from=build /app/bin /app/bin
ENTRYPOINT ["/app/bin"]
