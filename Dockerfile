FROM golang:1.11 AS builder


WORKDIR $GOPATH/src/github.com/Harnish/https-dns-proxy
COPY go.sum go.mod ./
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /app .

FROM scratch
WORKDIR /root/

COPY --from=builder ./app .
ENTRYPOINT ["./app"]

