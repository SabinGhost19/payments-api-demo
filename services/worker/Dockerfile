FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/payments-worker .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/payments-worker /payments-worker
USER nonroot:nonroot
ENTRYPOINT ["/payments-worker"]
