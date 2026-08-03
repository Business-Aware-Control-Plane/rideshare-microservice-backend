FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
ARG TARGETOS TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o build/trip-service ./services/trip-service/cmd/main.go

FROM alpine
WORKDIR /app
COPY --from=builder /app/build/trip-service /app/build/trip-service
ENTRYPOINT ["/app/build/trip-service"]