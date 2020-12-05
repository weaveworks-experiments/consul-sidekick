IMAGE=weaveworks/consul-sidekick
TAGGED_IMAGE=$(IMAGE):$(shell git rev-parse --abbrev-ref HEAD)-$(shell git rev-parse --short HEAD)

.PHONY: all push

all: Dockerfile consul-sidekick
	docker build -t $(IMAGE) .

consul-sidekick: main.go
	go build -ldflags "-extldflags \"-static\" -linkmode=external -s -w" .

push: all
	docker push $(IMAGE)
	docker tag $(IMAGE) $(TAGGED_IMAGE)
	docker push $(TAGGED_IMAGE)
