#!/bin/sh

# Create network if not exists
docker network inspect common >/dev/null 2>&1 || docker network create common

# Make sure the DB exists so the container can bind it.
touch feeds.db

# Start the app server if it exists (nop if already running), otherwise create it
docker start rsn-app || docker run -d \
	-v $PWD/feeds.db:/app/feeds.db \
	--restart unless-stopped \
	--network common \
	--name rsn-app \
	rsn-app

# Then the twitch bot
docker start twitch-bot || docker run -d \
	-v $PWD/Bot/token-master.json:/app/token-master.json \
	-v $PWD/Bot/token-bot.json:/app/token-bot.json \
	--restart unless-stopped \
	--network common \
	--name twitch-bot \
	twitch-bot

# And finally the NGINX proxy
docker start nginx || docker run -d -p 443:443 \
	-v $PWD/NGINX:/etc/nginx/ \
	-v $PWD/Site:/usr/share/nginx/html \
	--restart unless-stopped \
	--network common \
	--name nginx \
	nginx
