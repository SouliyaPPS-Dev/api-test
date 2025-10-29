FROM golang:1.23 AS builder

WORKDIR /app

# Ensure the required toolchain is available (Go >= 1.24 as per go.mod).
RUN go toolchain download go1.24.3

COPY go.mod go.sum ./
RUN GOTOOLCHAIN=go1.24.3 go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOTOOLCHAIN=go1.24.3 \
    go build -trimpath -buildvcs=false -o /bin/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /bin/server /app/server

EXPOSE 8080

ENTRYPOINT ["/app/server"]
