FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE
RUN test -n "$SERVICE" && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/service "./${SERVICE}"

FROM alpine:3.22

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /out/service ./service

USER 65532:65532
ENTRYPOINT ["./service"]
