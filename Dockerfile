FROM golang:1.16 as builder

WORKDIR /go/src/github.com/smpio/kube-nodeaffinity-fix/

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags "-s -w"


FROM gcr.io/distroless/static
COPY --from=builder /go/src/github.com/smpio/kube-nodeaffinity-fix/kube-nodeaffinity-fix /
ENTRYPOINT ["/kube-nodeaffinity-fix"]
