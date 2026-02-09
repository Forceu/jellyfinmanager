FROM golang:1.25.6-bookworm

## To compile:
## cd ./build/
## docker build . --tag jellyfinmanager-builder
## docker run --rm -it -v ../:/usr/src/myapp -w /usr/src/myapp jellyfinmanager-builder

RUN \
  apt-get update && \
  apt-get install -y ca-certificates openssl zip && \
  update-ca-certificates && \
  rm -rf /var/lib/apt

COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
