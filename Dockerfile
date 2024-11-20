FROM golang:1.21 as builder

WORKDIR /tmp/freemp3

COPY . .

RUN go mod download && \
    CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o freemp3 .

FROM node:20.14.0-alpine3.20 as producer

WORKDIR /data/freemp3

COPY . .
RUN npm install

COPY --from=builder /tmp/freemp3/freemp3 ./

RUN chmod a+x freemp3

CMD ["./freemp3"]
