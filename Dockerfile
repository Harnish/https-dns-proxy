FROM golang:1.11 AS builder


WORKDIR $GOPATH/src/github.com/Harnish/https-dns-proxy
ENV GO111MODULE=on
COPY go.sum go.mod ./
RUN go mod download

FROM builder AS server_builder
# Here we copy the rest of the source code

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /app .

FROM scratch
WORKDIR /root/

COPY --from=server_builder ./app .
ENTRYPOINT ["./app"]

