FROM golang:1.23 as builder
WORKDIR /backend
COPY go.mod go.sum ./
RUN go mod tidy && mkdir -p bin
COPY ./ ./
RUN go build -o bin/backend main.go
CMD ["bin/backend"]
