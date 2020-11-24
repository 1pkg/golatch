FROM golang:1.15

RUN mkdir -p $GOPATH/src/github.com/1pkg/go2close
WORKDIR $GOPATH/src/github.com/1pkg/go2close
ADD ./* ./
ADD ./vendor ./vendor

CMD ["go", "test", "-mod=vendor", "-v", "-race", "-count=1", "-coverprofile", "test.cover", "./..."]