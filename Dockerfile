FROM golang:1.25.2 AS builder

COPY ./cmd /app/cmd
COPY ./dev-scripts/build-release-server.sh /app/dev-scripts/build-release-server.sh
COPY ./internal /app/internal
COPY ./go.* /app/

WORKDIR /app

RUN ./dev-scripts/build-release-server.sh

FROM litestream/litestream:0.3.13 AS litestream

FROM alpine:3.15

RUN apk add --no-cache bash tzdata

ARG TZ
RUN if [[ -n "${TZ}" ]]; then \
      ln -snf "/usr/share/zoneinfo/${TZ}" /etc/localtime && \
      echo "${TZ}" > /etc/timezone; \
    fi

COPY --from=builder /app/release/dump /app/dump
COPY --from=litestream /usr/local/bin/litestream /app/litestream
COPY ./docker-entrypoint /app/docker-entrypoint
COPY ./litestream.yml /etc/litestream.yml

WORKDIR /app

ENTRYPOINT ["/app/docker-entrypoint"]
