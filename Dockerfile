FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /mailflow ./cmd/api/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

# Set the current working directory inside the container
WORKDIR /root/
COPY --from=builder /mailflow .
COPY --from=builder /app/.env.example .

EXPOSE 8080
CMD ["./mailflow"]