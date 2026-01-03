FROM golang:1.24-alpine AS go-builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM node:20-slim AS node-builder

WORKDIR /app/scripts/validate-strudel

# copy validator source and build standalone binary
COPY scripts/validate-strudel/package*.json ./
RUN npm install

COPY scripts/validate-strudel/*.js ./
RUN npm run build

FROM alpine:latest

WORKDIR /app

# copy Go binary and resources
COPY --from=go-builder /app/server .
COPY --from=go-builder /app/resources ./resources

# copy compiled validator binary
COPY --from=node-builder /app/scripts/validate-strudel/dist/validator-linuxstatic-x64 ./scripts/validate-strudel/validator-linuxstatic-x64

EXPOSE 8080

CMD ["./server"]