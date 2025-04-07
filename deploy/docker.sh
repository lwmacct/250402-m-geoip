#!/usr/bin/env bash

__main() {

  {
    # 镜像准备
    _image1="1181.s.kuaicdn.cn:11818/ghcr.io/lwmacct/250402-m-geoip:v0.0.0-x86_64"
    {
      _img_name=$(echo "$_image1" | cut -d ':' -f 1-2)
      _img_tag=$(skopeo list-tags "docker://$_img_name" | jq -r '.Tags[]' | sort -V | tail -n1)
      if [[ "$_img_tag" != "" ]]; then
        _image1="$_img_name:$_img_tag"
      fi
      echo "$_image1"
    }
    _image2=$(docker images -q "$_image1" 2>/dev/null)
    if [[ "$_image2" == "" ]]; then
      docker pull "$_image1"
      _image2=$(docker images -q "$_image1")
    fi
  }

  _apps_name="m-geoip-250402"
  _apps_data="/data/$_apps_name"
  if [[ ! -f "$_apps_data/.env" ]]; then
    mkdir -p "$_apps_data"
    touch "$_apps_data/.env"
  fi
  cat <<EOF | docker compose -p "$_apps_name" -f - up -d --remove-orphans
services:
 main:
   container_name: $_apps_name
   image: "$_image2"
   restart: always
   network_mode: host
   privileged: true
   volumes:
     - /etc/localtime:/etc/localtime:ro
     - $_apps_data:/apps/data
   env_file:
     - $_apps_data/.env
   environment:
     - TZ=Asia/Shanghai
     - ACF_LOG_LEVEL=8
     - ACF_LOG_FILE=/apps/data/run.log
     - ACF_APP_LISTEN_ADDR=0.0.0.0:8000

   command:
     - app
     - start
     - run
     - --label
     - "$_apps_name"
EOF
}

__main
