FROM golang:alpine AS builder
WORKDIR /app
COPY --exclude=client/ . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o server server/server.o

FROM scratch
WORKDIR /app
COPY --from=builder /app/server .
CMD ["./server"]
