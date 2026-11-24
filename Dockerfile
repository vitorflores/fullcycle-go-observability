FROM golang:1.27-alpine AS builder
ARG SERVICO
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ms ./cmd/${SERVICO}

FROM alpine:3.24
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/ms /app/ms
CMD ["/app/ms"]
