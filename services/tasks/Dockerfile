FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download || true

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/tasks ./cmd/server

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/bin/tasks /app/tasks

EXPOSE 8082

CMD ["/app/tasks"]