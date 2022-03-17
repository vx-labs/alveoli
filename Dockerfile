FROM golang:alpine as builder
ENV CGO_ENABLED=0
RUN mkdir -p $GOPATH/src/github.com/vx-labs
WORKDIR $GOPATH/src/github.com/vx-labs/alveoli
COPY go.* ./
RUN go mod download
COPY . ./
RUN go test ./...
ARG BUILT_VERSION="snapshot"
RUN go build -buildmode=exe -ldflags="-s -w -X github.com/vx-labs/alveoli/cmd/alveoli/version.BuiltVersion=${BUILT_VERSION}" \
       -a -o /bin/alveoli ./cmd/alveoli

FROM alpine:3.15 as prod
ENTRYPOINT ["/usr/bin/alveoli"]
RUN apk -U add ca-certificates && \
    rm -rf /var/cache/apk/*
COPY --from=builder /bin/alveoli /usr/bin/alveoli
