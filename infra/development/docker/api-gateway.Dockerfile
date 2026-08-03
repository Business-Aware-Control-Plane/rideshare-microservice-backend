FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
ARG TARGETOS TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o build/api-gateway ./services/api-gateway

FROM alpine
WORKDIR /app
COPY --from=builder /app/build/api-gateway /app/build/api-gateway
ENTRYPOINT ["/app/build/api-gateway"]