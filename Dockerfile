FROM golangci/golangci-lint:v1.38.0-alpine AS builder
WORKDIR /go/src/app
COPY . .
RUN go get -d ./...
RUN go test ./...
RUN golangci-lint run
RUN go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o ifk8s main.go

FROM scratch
COPY --from=builder /go/src/app/ifk8s /
ENTRYPOINT ["/ifk8s"]
