FROM golang:1.24 AS builder
WORKDIR /app
COPY main.go .
RUN go mod init mock-america && go mod tidy && CGO_ENABLED=0 go build -o mock-america .

FROM gcr.io/distroless/static
COPY --from=builder /app/mock-america /usr/local/bin/mock-america
ENTRYPOINT ["mock-america"]
