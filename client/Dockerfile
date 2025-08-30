FROM golang:alpine AS builder
WORKDIR /app
COPY --exclude=server/ . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o client client/client.o

FROM scratch
WORKDIR /app
COPY --from=builder /app/client .
CMD ["./client"]
