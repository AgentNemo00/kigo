FROM golang:1.22 as builder

ARG PRVT_KEY
ARG MAIN_PATH

WORKDIR /root

COPY ../.. .

RUN mkdir -p /root/.ssh
COPY $PRVT_KEY /root/.ssh/id_rsa
RUN chmod 700 /root/.ssh/id_rsa
RUN echo "Host github.com\n\tStrictHostKeyChecking no\n" >> /root/.ssh/config
RUN git config --global url.ssh://git@github.com/.insteadOf https://github.com/
RUN go mod download

RUN CGO_ENABLED=1 GOOS=linux go build -o app -a -ldflags '-linkmode external -extldflags "-static"' $MAIN_PATH

RUN rm -R /root/.ssh

FROM alpine:latest
RUN apk --no-cache add ca-certificates doas

ENV APP_CONTAINERIZATION=true
ENV USER=docker
ENV GROUPNAME=$USER
ENV UID=12345
ENV GID=23456

RUN addgroup \
    --gid "$GID" \
    "$GROUPNAME" \
&&  adduser \
    --disabled-password \
    --gecos "" \
    --home "/home/docker" \
    --ingroup "$GROUPNAME" \
    --no-create-home \
    --uid "$UID" \
    $USER

RUN echo 'permit nopass docker as root cmd /home/docker/app' > /etc/doas.d/doas.conf
RUN doas -C /etc/doas.d/doas.conf && echo "config ok" || exit 1

COPY --from=builder /root/app /home/docker/app
RUN chown -R docker:docker /home/docker

USER $USER

CMD ["/bin/sh"]
