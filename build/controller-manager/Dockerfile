FROM golang:1.24.0 AS builder

WORKDIR /kantaloupe

ENV GO111MODULE on
ENV CGO_ENABLED 0
ENV GOOS linux

COPY . .

RUN go mod tidy
RUN go build -ldflags="-s -w" -o kantaloupe-controller-manager ./cmd/controller-manager/main.go && echo "Building GOARCH of $GOARCH.."


# copy kantaloupe controller manager from builder
FROM  alpine:3.18

COPY --from=builder /kantaloupe/kantaloupe-controller-manager /bin/

CMD ["/bin/kantaloupe-controller-manager"]
