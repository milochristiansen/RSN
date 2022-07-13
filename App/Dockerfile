
FROM golang:1.18-alpine3.16 AS build-go

WORKDIR /app

RUN apk add --no-cache build-base

COPY . .

RUN go build -o server.bin

########################################################################################################################

FROM alpine:3.16

WORKDIR /app

COPY --from=build-go /app/server.bin .

ENTRYPOINT ["/app/server.bin"]
