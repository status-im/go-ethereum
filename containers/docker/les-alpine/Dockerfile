FROM alpine:3.4

RUN \
  apk add --update go git make gcc musl-dev openssh-client && \

  # add SSH key
  wget -O /tmp/id_rsa http://192.168.99.1:8080/status_go_id_rsa && \
  chmod 600 /tmp/id_rsa && \
  eval $(ssh-agent) && \
  echo -e "StrictHostKeyChecking no" >> /etc/ssh/ssh_config && \
  ssh-add /tmp/id_rsa && \

  # clone status-go
  mkdir -p /usr/lib/go/src/github.com/status-im && \
  git clone --depth 1 --branch develop git@github.com:farazdagi/status-go.git /usr/lib/go/src/github.com/status-im/status-go && \

  # clone go-ethereum (and install everything)
  git clone --depth 1 --branch status-light-stable https://github.com/status-im/go-ethereum && \
  (cd go-ethereum && make geth) && \
  cp go-ethereum/build/bin/geth /geth && \
  apk del go git make gcc musl-dev openssh-client && \
  rm -rf /go-ethereum && rm -rf /var/cache/apk/* && rm /tmp/id_rsa

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/geth"]
