#!/bin/sh

docker image save rsn-app:latest | ssh root@ssh.httpscolonslashslashwww.com docker image load
docker image save twitch-bot:latest | ssh root@ssh.httpscolonslashslashwww.com docker image load
