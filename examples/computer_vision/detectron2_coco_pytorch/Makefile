TAG := determinedai/example-detectron2:0.6-cuda-10.2-pytorch-1.10

.PHONY: build
build:
	docker build -f Dockerfile -t $(TAG) . && \
	docker push $(TAG)
