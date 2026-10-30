# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o parser .

# Run stage
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/parser .
COPY grammar.ebnf .
COPY robust-go-parser.zip .
COPY testdata/ ./testdata/

EXPOSE 8080

CMD ["./parser", "-serve", ":8080"]
