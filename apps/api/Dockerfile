FROM golang:1.25.2-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linu go build -ldflags="-w -s" -o main .

FROM gcr.io/distroless/static-debian12:latest

RUN adduser -D appuser
USER appuser

WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]