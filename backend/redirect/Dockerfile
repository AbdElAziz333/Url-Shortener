#-----------------Build Stage--------------
FROM golang:1.26.3-alpine3.23 AS builder

WORKDIR /app

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o app ./cmd

#---------Runtime Stage--------------

FROM scratch

WORKDIR /app

COPY --from=builder /app/app .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8082

CMD ["./app"]
