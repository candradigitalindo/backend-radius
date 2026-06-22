FROM golang:1.25-alpine AS builder

WORKDIR /build

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o app cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o migrate cmd/migrate/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o worker cmd/worker/main.go

# Runtime
FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata iproute2 wireguard-tools
ENV TZ=Asia/Jakarta

COPY --from=builder /build/app .
COPY --from=builder /build/migrate .
COPY --from=builder /build/worker .
COPY --from=builder /build/static ./static
COPY --from=builder /build/docs/swagger.json ./docs/swagger.json
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

EXPOSE 3000

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["./app"]
