FROM golang:1.16-alpine AS builder

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
RUN apk add --no-cache iptables

ENTRYPOINT [ "/usr/local/bin/sshtunnel", "daemon", "--logfile", "/dev/stdout" ]