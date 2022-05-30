# get mcutils binary
FROM golang as mcutils

RUN go install github.com/xrjr/mcutils/cmd/mcutils@latest

# get minecraft bedrock server
FROM debian:bullseye as build

ARG MC_VER=1.19.10.20

ENV DEBIAN_FRONTEND=noninteractive
ENV MINECRAFT_VER=${MC_VER}

WORKDIR /build

RUN apt update && apt upgrade -y \
    && apt install -y curl unzip

RUN curl -O https://minecraft.azureedge.net/bin-linux/bedrock-server-${MINECRAFT_VER}.zip \
    && unzip -qq bedrock-server-${MINECRAFT_VER}.zip \
    && rm bedrock-server-${MINECRAFT_VER}.zip \
    && echo ${MINECRAFT_VER} > /build/version

# put it all together
FROM debian:bullseye

ENV DEBIAN_FRONTEND=noninteractive
ENV APP_DIR=/opt/minecraft

RUN apt update && apt upgrade -y \
    && apt autoremove -y \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -d ${APP_DIR} -s /bin/bash minecraft \
    && chown -R minecraft ${APP_DIR}

WORKDIR ${APP_DIR}

COPY --from=mcutils /go/bin/mcutils /usr/local/bin/mcutils
COPY --from=build /build ${APP_DIR}

COPY docker-entrypoint.sh ${APP_DIR}/docker-entrypoint.sh

RUN chmod +x ${APP_DIR}/docker-entrypoint.sh

CMD ${APP_DIR}/docker-entrypoint.sh