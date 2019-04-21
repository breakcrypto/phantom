FROM golang:1.12.4

WORKDIR /go/src/phantom
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

RUN go build

CMD ["build.sh"]