# syntax=docker/dockerfile:1.7-labs
FROM golang:alpine AS builder
WORKDIR /app
COPY --exclude=client/ . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o server server/server.go

FROM alpine
WORKDIR /app
COPY --from=builder /app/server .
ADD bd .
CMD ["./server"]
