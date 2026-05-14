
FROM golang:1.26-alpine3.22 AS build-go

WORKDIR /App

COPY . .

RUN go build -o server.bin

########################################################################################################################

FROM alpine:3.22

WORKDIR /App

COPY --from=build-go /App/server.bin .

ENTRYPOINT ["/App/server.bin"]
