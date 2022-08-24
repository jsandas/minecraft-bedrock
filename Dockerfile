# get minecraft bedrock server
FROM debian:bullseye as build

ARG MC_VER=1.19.21.01

ENV DEBIAN_FRONTEND=noninteractive
ENV MINECRAFT_VER=${MC_VER}

WORKDIR /build

RUN apt-get update && apt-get upgrade -y \
    && apt-get install -y curl unzip
RUN curl -O https://minecraft.azureedge.net/bin-linux/bedrock-server-${MINECRAFT_VER}.zip
RUN unzip -qq bedrock-server-${MINECRAFT_VER}.zip
RUN rm bedrock-server-${MINECRAFT_VER}.zip
RUN echo ${MINECRAFT_VER} > /build/version

# put it all together
FROM debian:bullseye

ENV DEBIAN_FRONTEND=noninteractive
ENV APP_DIR=/opt/minecraft

RUN apt-get update && apt-get upgrade -y \
    && apt-get install -y curl \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/*
RUN useradd -m -d ${APP_DIR} -s /bin/bash minecraft \
    && chown -R minecraft ${APP_DIR}

WORKDIR ${APP_DIR}

COPY --from=itzg/mc-monitor /mc-monitor /usr/local/bin/mc-monitor
COPY --from=build /build ${APP_DIR}
COPY docker-entrypoint.sh ${APP_DIR}
COPY mccli ${APP_DIR}

RUN chmod +x ${APP_DIR}/docker-entrypoint.sh \
    && chmod +x ${APP_DIR}/mccli \
    && chown -R minecraft /opt/minecraft

USER minecraft

ENTRYPOINT ["/opt/minecraft/docker-entrypoint.sh"]
