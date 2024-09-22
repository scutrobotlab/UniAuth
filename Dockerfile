FROM --platform=$BUILDPLATFORM node:20 AS FRONT
WORKDIR /web
COPY ./web .
RUN yarn install --frozen-lockfile --network-timeout 1000000 && yarn run build


FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS BACK
WORKDIR /go/src/casdoor
RUN apk add --no-cache git
ADD go.mod go.sum ./
RUN go mod download

COPY . .
RUN sh ./build.sh

FROM alpine:3.16
LABEL MAINTAINER="https://casdoor.org/"
ARG USER=casdoor
ARG TARGETOS
ARG TARGETARCH
ENV BUILDX_ARCH="${TARGETOS:-linux}_${TARGETARCH:-amd64}"
ENV TZ=Asia/Shanghai

RUN apk add --no-cache tzdata curl ca-certificates sudo && update-ca-certificates

RUN adduser -D $USER -u 1000 \
    && echo "$USER ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/$USER \
    && chmod 0440 /etc/sudoers.d/$USER \
    && mkdir logs \
    && chown -R $USER:$USER logs

USER 1000
WORKDIR /
COPY --from=BACK --chown=$USER:$USER /go/src/casdoor/server_${BUILDX_ARCH} ./server
COPY --from=BACK --chown=$USER:$USER /go/src/casdoor/swagger ./swagger
COPY --from=BACK --chown=$USER:$USER /go/src/casdoor/conf/app.conf ./conf/app.conf
COPY --from=FRONT --chown=$USER:$USER /web/build ./web/build

ENTRYPOINT ["/server"]