FROM golang:1.18 AS build

WORKDIR /dockervm

ENV CGO_ENABLED=0

ENV GOPROXY=https://goproxy.cn,direct

COPY . /dockervm

RUN go build -mod=vendor -o /bin/startvm

FROM ubuntu:20.04

# label build time, git branch and git log
ARG BUILD_TIME=unspecified
ARG GIT_BRANCH=unspecified
ARG GIT_COMMIT=unspecified

LABEL build_time=$BUILD_TIME
LABEL git_branch=$GIT_BRANCH
LABEL git_commit=$GIT_COMMIT

# using docker api to passing time to docker
# enable below code when docker-vm enable macOS or windows
ENV TZ "Asia/Shanghai"

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y tzdata && \
    echo $TZ > /etc/timezone && \
    ln -fs /usr/share/zoneinfo/$TZ /etc/localtime && \
    dpkg-reconfigure tzdata -f noninteractive

COPY --from=build /bin/startvm /bin/startvm

ENTRYPOINT ["/bin/startvm"]
