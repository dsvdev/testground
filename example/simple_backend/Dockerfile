FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /simple_backend ./example/simple_backend/cmd

FROM alpine:latest
COPY --from=builder /simple_backend /simple_backend
EXPOSE 8080
CMD ["/simple_backend"]