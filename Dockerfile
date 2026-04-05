FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go mod download && CGO_ENABLED=0 go build -o portfolio ./cmd/portfolio/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/portfolio .
ENV PORT=9808 DATA_DIR=/data
EXPOSE 9808
CMD ["./portfolio"]
