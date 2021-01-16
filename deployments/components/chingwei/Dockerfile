FROM alpine:3.12

RUN apk --update add git less openssh && \
    rm -rf /var/lib/apt/lists/* && \
    rm /var/cache/apk/*

ADD chingwei /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/chingwei"]