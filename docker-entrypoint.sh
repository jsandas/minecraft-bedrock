#!/bin/bash

APP_DIR=${APP_DIR:-/opt/minecraft}

function config() {
    _envvars=$(env | egrep '^CFG_' | sed 's/CFG_//g' | sed 's/_/-/g')
    for _envvar in ${_envvars[@]}
    do
        _key=$(echo ${_envvar} | awk -F '=' '{print $1}' | tr '[:upper:]' '[:lower:]') 
        _val=$(echo ${_envvar} | awk -F '=' '{print $2}')

        echo " Updating $_key..."
        sed -i 's/'"${_key}"'=.*/'"${_key}"'='"${_val}"'/g' ${APP_DIR}/server.properties
    done
}

function download() {
    curl -O https://minecraft.azureedge.net/bin-linux/bedrock-server-${MINECRAFT_VER}.zip
    unzip -qq bedrock-server-${MINECRAFT_VER}.zip
    rm bedrock-server-${MINECRAFT_VER}.zip
}

if [[ "$@" == "" ]]; then
    if [[ $EULA_ACCEPT != 'true' ]]; then
        echo " Please accept the Minecraft EULA and Microsoft Privacy Policy"
        echo " with env var EULA_ACCEPT=true"
        echo " Links:"
        echo "   https://www.minecraft.net/en-us/terms"
        echo "   https://privacy.microsoft.com/en-us/privacystatement"
        echo
        exit 1
    fi

    download 

    config

    export LD_LIBRARY_PATH=.
    # create named pipe for streaming input from another shell
    mkfifo input_pipe
    # create file for streaming output to another shell
    touch output_pipe
    tail -f input_pipe | ./bedrock_server 2>&1 | tee output_pipe
fi

exec "$@"
