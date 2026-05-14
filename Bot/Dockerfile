
FROM golang:1.26-alpine3.22 AS build-go

WORKDIR /Bot

COPY . .

RUN go build -o server.bin

########################################################################################################################

FROM alpine:3.22

WORKDIR /Bot

COPY --from=build-go /Bot/server.bin .

ENTRYPOINT ["/Bot/server.bin"]
