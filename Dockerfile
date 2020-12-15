FROM golang:alpine as build

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

RUN apk --no-cache add ca-certificates

WORKDIR $GOPATH/src/fp-dim-egress-smc-go/

RUN mkdir certs

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -ldflags="-w -s" -o /go/bin/fp-smc

FROM scratch AS release

COPY --from=build /go/src/fp-dim-egress-smc-go/certs/ /etc/ssl/certs/
COPY --from=build /go/bin/fp-smc /
COPY --from=build /go/src/fp-dim-egress-smc-go/config/ /config/


ENTRYPOINT ["/fp-smc"]