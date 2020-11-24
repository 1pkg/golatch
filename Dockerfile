FROM golang:1.15

RUN mkdir -p $GOPATH/src/github.com/1pkg/golock
WORKDIR $GOPATH/src/github.com/1pkg/golock
ADD ./* ./
ADD ./vendor ./vendor

CMD ["go", "test", "-mod=vendor", "-v", "-race", "-count=1", "-coverprofile", "test.cover", "./..."]