FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /report-service ./main.go

FROM alpine:3.21
RUN adduser -D -u 1000 appuser
COPY --from=builder /report-service /report-service
USER appuser
EXPOSE 8083
ENTRYPOINT ["/report-service"]
