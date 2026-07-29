# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /server .

# Runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /server /app/server
EXPOSE 8080
ENV PORT=8080
ENV DB_PATH=/data/inventory.db
VOLUME ["/data"]
CMD ["/app/server"]