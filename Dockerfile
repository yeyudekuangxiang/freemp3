FROM golang:1.21 as builder

WORKDIR /tmp/freemp3

COPY . .
RUN ls
RUN go mod download && \
    CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o freemp3 .
RUN ls    

FROM node:20.14.0 as producer

WORKDIR /data/freemp3

COPY . .
RUN ls
RUN npm install

RUN apt-get update && apt-get install -y \
     wget \
     --no-install-recommends \
     && apt-get install -y \
     ca-certificates \
     fonts-liberation \
     libappindicator3-1 \
     libasound2 \
     libatk-bridge2.0-0 \
     libatk1.0-0 \
     libcups2 \
     libdbus-1-3 \
     libdrm2 \
     libgbm1 \
     libnspr4 \
     libnss3 \
     libxcomposite1 \
     libxdamage1 \
     libxrandr2 \
     xdg-utils \
     && rm -rf /var/lib/apt/lists/*

COPY --from=builder /tmp/freemp3/freemp3 ./

RUN chmod a+x freemp3

CMD ["./freemp3"]
