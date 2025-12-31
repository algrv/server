FROM golang:1.24.5

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -o server ./cmd/server

EXPOSE 8080

CMD ["./server"]