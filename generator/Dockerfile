FROM golang:1.25
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o generator
CMD ["./generator"]
