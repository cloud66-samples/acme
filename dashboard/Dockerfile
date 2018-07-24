FROM golang:1.10

WORKDIR /go/src/dashboard
COPY main.go .
RUN go get -d -v ./...
RUN go build

RUN mkdir static
COPY static/. static/.

CMD ["/go/src/dashboard/dashboard"]
