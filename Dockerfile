FROM debian:bookworm AS builder

ENV WRAPPER_VERSION=0.9.1

RUN apt-get update && apt-get install -y curl \
    && curl -fssL -O https://github.com/jsandas/minecraft-bedrock-wrapper/releases/download/v${WRAPPER_VERSION}/minecraft-bedrock-wrapper_${WRAPPER_VERSION}_linux_amd64.tar.gz

RUN tar -xzf minecraft-bedrock-wrapper_${WRAPPER_VERSION}_linux_amd64.tar.gz \
    && rm minecraft-bedrock-wrapper_${WRAPPER_VERSION}_linux_amd64.tar.gz


FROM debian:bookworm

ARG MINECRAFT_VERSION=1.26.61.1

ENV DEBIAN_FRONTEND=noninteractive
ENV APP_DIR=/opt/minecraft
ENV MINECRAFT_VERSION=${MINECRAFT_VERSION}
ENV PORT=19132

# Minecraft bedrock server requires libcurl
RUN apt-get update && apt-get upgrade -y \
    && apt-get install -y ca-certificates \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -d ${APP_DIR} -s /bin/bash minecraft \
    && chown -R minecraft ${APP_DIR}

WORKDIR ${APP_DIR}

COPY --from=builder /minecraft-bedrock-wrapper /minecraft-bedrock-wrapper
COPY --from=itzg/mc-monitor /mc-monitor /usr/local/bin/mc-monitor

RUN chown -R minecraft ${APP_DIR}

USER minecraft

EXPOSE 19132
EXPOSE 19132/udp
EXPOSE 19133/udp
EXPOSE 7551/udp

ENTRYPOINT ["/minecraft-bedrock-wrapper"]
