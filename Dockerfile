FROM ubuntu:impish

ENV DEBIAN_FRONTEND=noninteractive
ENV APP_DIR=/opt/minecraft

RUN useradd -d ${APP_DIR} -s /bin/bash minecraft 

RUN apt update && apt upgrade -y \
    && apt install -y curl unzip \
    && rm -rf /var/lib/apt/lists/*

WORKDIR ${APP_DIR}

COPY docker-entrypoint.sh ${APP_DIR}/docker-entrypoint.sh

RUN chmod +x ${APP_DIR}/docker-entrypoint.sh

CMD ${APP_DIR}/docker-entrypoint.sh