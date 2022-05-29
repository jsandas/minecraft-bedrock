#!/bin/bash

APP_DIR=${APP_DIR:-/opt/minecraft}
MINECRAFT_VER=$(curl -s -H "accept-language:*" -H "User-Agent: Mozilla/5.0" https://www.minecraft.net/en-us/download/server/bedrock | grep serverBedrockLinux | grep -o -P '(?<=bedrock-server-).*(?=.zip)')
DOWNLOAD_URL="https://minecraft.azureedge.net/bin-linux/bedrock-server-$MINECRAFT_VER.zip"

function config() {
    _envvars=$(env | egrep '^CFG_' | sed 's/CFG_//g' | sed 's/_/-/g')
    for _envvar in ${_envvars[@]}
    do
        _key=$(echo $_envvar | awk -F '=' '{print $1}' | tr '[:upper:]' '[:lower:]') 
        _val=$(echo $_envvar | awk -F '=' '{print $2}')

        echo " Updating $_key..."
        sed -i 's/'"${_key}"'=.*/'"${_key}"'='"${_val}"'/g' $APP_DIR/server.properties
    done
}

if [[ ! -d $MINECRAFT_VER ]]; then
    echo "Installing minecraft version: $MINECRAFT_VER"

    # Download and decompress archive
    curl -s -O $DOWNLOAD_URL
    unzip -qq bedrock-server-$MINECRAFT_VER.zip
    rm bedrock-server-$MINECRAFT_VER.zip
fi

config

echo "Starting minecraft server..."
LD_LIBRARY_PATH=. ./bedrock_server