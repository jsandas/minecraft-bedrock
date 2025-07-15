FROM golang:1.24 AS builder

WORKDIR /app

COPY . .

RUN go build -o bedrock-server-wrapper ./cmd/bedrock-server-wrapper


FROM debian:bookworm

ARG MC_VER=1.21.94.2

ENV DEBIAN_FRONTEND=noninteractive
ENV MINECRAFT_VER=${MC_VER}
ENV APP_DIR=/opt/minecraft

# Minecraft bedrock server requires libcurl
RUN apt-get update && apt-get upgrade -y \
    && apt-get install -y curl \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -d ${APP_DIR} -s /bin/bash minecraft \
    && chown -R minecraft ${APP_DIR}

WORKDIR ${APP_DIR}

COPY --from=builder /app/bedrock-server-wrapper ${APP_DIR}/bedrock-server-wrapper
COPY --from=itzg/mc-monitor /mc-monitor /usr/local/bin/mc-monitor

RUN chown -R minecraft ${APP_DIR}

USER minecraft

EXPOSE 19132/udp
EXPOSE 19133/udp

ENTRYPOINT ["/opt/minecraft/bedrock-server-wrapper"]
