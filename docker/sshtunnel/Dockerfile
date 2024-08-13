FROM golang:1.23-alpine AS builder

RUN mkdir /src
WORKDIR /src
COPY go.mod go.sum main.go commands control dialer logger originaldest proxy rules server /src
COPY commands /src/commands
COPY control /src/control
COPY dialer /src/dialer
COPY logger /src/logger
COPY originaldest /src/originaldest
COPY proxy /src/proxy
COPY rules /src/rules
COPY server /src/server
RUN go build -o ./sshtunnel main.go

FROM alpine

COPY --from=builder /src/sshtunnel /usr/local/bin/sshtunnel
COPY docker/scripts/start-sshtunnel /usr/local/bin/start-sshtunnel
COPY docker/scripts/update-iptables /usr/local/bin/update-iptables
RUN apk add --no-cache iptables

ENTRYPOINT [ "/usr/local/bin/start-sshtunnel" ]