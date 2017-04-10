FROM alpine:3.5
MAINTAINER Weaveworks Inc <help@weave.works>
LABEL works.weave.role=system
WORKDIR /home/weave
COPY ./consul-sidekick /home/weave/
ENTRYPOINT ["/home/weave/consul-sidekick"]
