# Multi-stage build: Go static binary on distroless.
# distroless/static-debian12:nonroot contains no shell, no package manager,
# nothing executable beyond our binary -> Trivy/SBOM report effectively
# nothing to flag. Used intentionally as the "clean Verified" sample.
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY main.go ./
# CGO disabled so the binary is fully static and runs on distroless/static.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/payments-api .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/payments-api /payments-api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/payments-api"]
