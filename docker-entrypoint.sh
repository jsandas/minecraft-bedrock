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

config

echo "Starting minecraft server version: $(cat $APP_DIR/version)..."
LD_LIBRARY_PATH=. ./bedrock_server