FROM golang:1.23 as builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o krangctl ./cmd/krangctl
RUN CGO_ENABLED=0 go build -o krangd ./cmd/krangd

FROM debian:stable-slim
COPY --from=builder /app/krangd /krangd
ENTRYPOINT ["/krangd"]
