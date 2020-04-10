# Builder
FROM golang:1.14.1-alpine3.11 AS builder
WORKDIR /build
COPY src ./
RUN go build -o main .

# APP
FROM alpine:3.11
WORKDIR /app
COPY  --from=builder /build/main ./
EXPOSE 4443
CMD ["./main"]