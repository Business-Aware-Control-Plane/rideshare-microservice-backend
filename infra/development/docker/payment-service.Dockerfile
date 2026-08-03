FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
ARG TARGETOS TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o build/payment-service ./services/payment-service/cmd/main.go

FROM alpine
WORKDIR /app
COPY --from=builder /app/build/payment-service /app/build/payment-service
ENTRYPOINT ["/app/build/payment-service"]