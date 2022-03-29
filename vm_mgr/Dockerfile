FROM golang:1.15-alpine AS build

WORKDIR /dockervm

ENV CGO_ENABLED=0

ENV GOPROXY=https://goproxy.cn,direct

COPY . /dockervm

RUN go build -mod=vendor -o /bin/startvm

FROM ubuntu:20.04

# using docker api to passing time to docker
# enable below code when docker-vm enable macOS or windows
ENV TZ "Asia/Shanghai"

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y tzdata && \
    echo $TZ > /etc/timezone && \
    ln -fs /usr/share/zoneinfo/$TZ /etc/localtime && \
    dpkg-reconfigure tzdata -f noninteractive

RUN mkdir -p /mount/contracts && mkdir -p /mount/sock

COPY --from=build /bin/startvm /bin/startvm

ENTRYPOINT ["/bin/startvm"]