FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cluster-server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder --chown=nonroot:nonroot /out/cluster-server /cluster-server
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/cluster-server"]
