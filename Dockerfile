FROM golang:1.19.2 AS builder
WORKDIR /go/src/service
COPY . .
RUN go mod tidy
RUN go build mock_service

EXPOSE 8080

ENTRYPOINT ["/go/src/service/mock_service"]