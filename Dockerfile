# Build Stage
FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o main .

# Runtime Stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
RUN chmod a+x /app/main && apk add --update alpine-conf tzdata  && setup-timezone -z Asia/Jakarta
CMD ["/app/main"]
