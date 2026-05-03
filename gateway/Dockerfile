#-----------------Build Stage--------------
FROM golang:1.25.6-alpine AS builder

WORKDIR /app

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

COPY go.mod .

WORKDIR /app/shortener

RUN go mod download
RUN go build -o app ./cmd/shortener

#---------Runtime Stage--------------

FROM scratch

WORKDIR /app

COPY --from=builder /app/app .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080

CMD ["./app"]
