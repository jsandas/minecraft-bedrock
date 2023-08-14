#!/bin/bash

MC_VER=$(curl -s -H "accept-language:*" -H "User-Agent: Mozilla/5.0" https://www.minecraft.net/en-us/download/server/bedrock | grep serverBedrockLinux | grep -o -P '(?<=bedrock-server-).*(?=.zip)')
IMAGE_VER=$(docker run ghcr.io/jsandas/minecraft-bedrock:latest cat version)
CHART_VER=$(cat charts/minecraft-bedrock/Chart.yaml | grep appVersion | awk '{print $2}' | tr -d '"')

if [[ -z $MC_VER || -z $IMAGE_VER || -z $CHART_VER ]]; then
    echo " Failed to retrieve all versions"
    echo " mc_ver=$MC_VER image_ver=$IMAGE_VER chart_ver=$CHART_VER"
    exit 1
fi

if [[ "$MC_VER" != "$IMAGE_VER" || "$MC_VER" != "$CHART_VER" ]]; then 
    echo " New version found: $MC_VER"
    sed -i 's/MC_VER=.*/MC_VER='"$MC_VER"'/g' Dockerfile
    sed -i 's/^appVersion:.*/appVersion: \"'"$MC_VER"'\"/g' charts/minecraft-bedrock/Chart.yaml
fi
