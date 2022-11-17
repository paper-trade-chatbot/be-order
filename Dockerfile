FROM alpine:latest

RUN apk add --update-cache tzdata
COPY be-order /be-order

ENTRYPOINT ["/be-order"]


