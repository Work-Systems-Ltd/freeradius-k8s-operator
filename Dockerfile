FROM docker.io/library/golang:1.22-alpine AS builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /operator ./cmd/operator/...

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /operator /operator

USER 65532:65532

ENTRYPOINT ["/operator"]
