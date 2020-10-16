DOCKER_REGISTRY ?= 614608043005.dkr.ecr.eu-central-1.amazonaws.com
DOCKER_IMAGE_TAG ?= latest 
DOCKER_FILE ?= Dockerfile
DOCKER_AWS_REGION ?= eu-central-1

.PHONY: docker-build
docker-build:
	docker build --no-cache -t $(DOCKER_REGISTRY)/$(SERVICE):$(DOCKER_IMAGE_TAG) -f $(DOCKER_FILE) .

.PHONY: push-image
push-image: docker-build
	aws ecr get-login-password --region $(DOCKER_AWS_REGION) | docker login --username AWS --password-stdin $(DOCKER_REGISTRY)
	docker push $(DOCKER_REGISTRY)/$(SERVICE):$(DOCKER_IMAGE_TAG)
