SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
include $(SELF_DIR)/common.mk

AWS_REGION ?= eu-central-1

.PHONY: ecr
ecr: guard-SERVICE guard-AWS_REGION 
ecr: ${SELF_DIR}/assets/ecr.yml
	aws cloudformation deploy \
		--no-fail-on-empty-changeset \
		--template-file $< \
		--stack-name $(SERVICE)-ecr \
		--parameter-overrides RepositoryName=$(SERVICE) \
		--region $(AWS_REGION)
