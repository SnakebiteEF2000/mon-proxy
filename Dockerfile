FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o mon-proxy .

FROM scratch

COPY --from=builder /app/mon-proxy /mon-proxy

ENTRYPOINT ["/mon-proxy"]
