FROM golang:1.24.0 AS builder

WORKDIR /kantaloupe

ENV GO111MODULE on
ENV CGO_ENABLED 0
ENV GOOS linux

COPY . .

RUN go mod tidy
RUN go build -ldflags="-s -w" -o kantaloupe-apiserver ./cmd/apiserver/main.go && echo "Building GOARCH of $GOARCH.."

# copy kantaloupe apiserver from builder
FROM alpine:3.18

COPY --from=builder /kantaloupe/kantaloupe-apiserver /bin/
COPY --from=builder /kantaloupe/api/assets /api/assets

CMD ["/bin/kantaloupe-apiserver"]