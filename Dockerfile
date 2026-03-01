FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go        ./
COPY lib/        ./lib/
COPY static/     ./static/

RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" -o /aggregator .

FROM scratch

COPY --from=builder /aggregator /aggregator

ENTRYPOINT ["/aggregator"]
