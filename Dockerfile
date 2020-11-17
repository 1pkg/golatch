FROM golang:1.15.3

COPY main.go main.go
RUN go get bou.ke/monkey
RUN go build -o /var/main main.go

ENTRYPOINT [ "/var/main" ]