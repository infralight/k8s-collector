FROM golangci/golangci-lint:v1.38.0-alpine AS builder
RUN apk --update add ca-certificates
WORKDIR /go/src/app
COPY . .
RUN go get -d ./...
RUN go test ./...
RUN golangci-lint run
RUN CGO_ENABLED=0 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o ifk8s main.go

FROM scratch
COPY --from=builder /go/src/app/ifk8s /
COPY --from=builder /etc/ssl/certs /etc/ssl/certs
ENTRYPOINT ["/ifk8s"]
