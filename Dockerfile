FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/fdt .

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /out/fdt /app/fdt
COPY FH4_packetformat.dat FM7_packetformat.dat /app/

EXPOSE 9999/udp
EXPOSE 8080

ENTRYPOINT ["/app/fdt"]
