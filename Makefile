IMAGE=weaveworks/consul-sidekick
QUAY_IMAGE=quay.io/$(IMAGE):$(shell git rev-parse --abbrev-ref HEAD)-$(shell git rev-parse --short HEAD)

.PHONY: all push

all: Dockerfile consul-sidekick
	docker build -t weaveworks/consul-sidekick .

consul-sidekick: main.go
	go build -ldflags "-extldflags \"-static\" -linkmode=external -s -w" .

push: all
	docker push $(IMAGE)
	docker tag $(IMAGE) $(QUAY_IMAGE)
	docker push $(QUAY_IMAGE)
