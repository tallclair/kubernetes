FROM alpine

ADD webhook webhook

ENTRYPOINT ["/webhook"]
