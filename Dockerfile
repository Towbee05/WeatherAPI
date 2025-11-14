# build stage
FROM golang:1.25 AS builder

WORKDIR /app

COPY go.sum go.mod ./

RUN go mod download

COPY . .

RUN go build -o server .

# run stage
FROM debian:latest
WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 8080

RUN ["./server"]