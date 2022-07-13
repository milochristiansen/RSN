#!/bin/sh

docker network create common

touch feeds.db
docker run -d \
	-v $PWD/feeds.db:/app/feeds.db \
	--restart unless-stopped \
	--network common \
	--name rsn-app \
	rsn-app

docker run -d -p 443:443 \
	-v $PWD/NGINX:/etc/nginx/ \
	-v $PWD/Site:/usr/share/nginx/html \
	--restart unless-stopped \
	--network common \
	--name nginx \
	nginx
