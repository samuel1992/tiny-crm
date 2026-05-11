FROM golang:1.24-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o tinycrm .

FROM alpine:3.21
ENV TZ=America/Sao_Paulo
RUN apk add --no-cache \
    ca-certificates tzdata \
    chromium nss freetype harfbuzz ttf-freefont
WORKDIR /app
COPY --from=builder /app/tinycrm ./
EXPOSE 8080
CMD ["./tinycrm"]
