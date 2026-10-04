FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o servico-binario .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/servico-binario .

EXPOSE 8082

CMD ["./servico-binario"]