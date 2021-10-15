FROM golang:1.15-alpine AS build

WORKDIR /dockervm

ENV CGO_ENABLED=0

ENV GOPROXY=https://goproxy.cn,direct

COPY . /dockervm

RUN go build -mod=vendor -o /bin/startvm

FROM ubuntu:20.04

COPY --from=build /bin/startvm /bin/startvm

ENTRYPOINT ["/bin/startvm"]