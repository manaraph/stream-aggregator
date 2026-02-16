FROM golang:1.25.7-alpine AS builder
WORKDIR /app
ENV GO111MODULE=on

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o /bin/generator ./generator

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /bin/generator /bin/generator
ENTRYPOINT ["/bin/generator"]
