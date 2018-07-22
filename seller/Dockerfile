FROM golang:1.10

WORKDIR /go/src/seller
COPY main.go .
RUN go get -d -v ./...
RUN go install -v ./...

CMD ["seller"]
