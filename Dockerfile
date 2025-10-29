FROM golang:1.23 AS builder

WORKDIR /app

# Pin the desired Go toolchain; the go command will fetch it on first use.
ENV GOTOOLCHAIN=go1.24.3

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -buildvcs=false -o /bin/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /bin/server /app/server

EXPOSE 8080

ENTRYPOINT ["/app/server"]
