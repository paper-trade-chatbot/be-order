PROJECT_ID=lisyaoran51
OS=$(shell uname | tr '[:upper:]' '[:lower:]')
SERVICE_NAME=$(shell basename `git rev-parse --show-toplevel`)
IMPORT_PATH=github.com/lisyaoran51/${SERVICE_NAME}
PROTOS=$(shell ls ./rf-protos/models/)
GIT_COMMIT_HASH=$(shell git rev-parse HEAD | cut -c -16)
BUILD_TIME=$(shell date +%s)
LDFLAGS = -X ${IMPORT_PATH}/global.ServiceName=${SERVICE_NAME}
LDFLAGS += -X ${IMPORT_PATH}/global.GitCommitHash=${GIT_COMMIT_HASH}
LDFLAGS += -X ${IMPORT_PATH}/global.BuildTime=${BUILD_TIME}
TAG=${PROJECT_ID}/${SERVICE_NAME}:${GIT_COMMIT_HASH}
IMAGE=lisyaoran51/${TAG}
ifeq (${OS}, darwin)
	SED_INPLACE = sed -i'.orig' -e
endif
ifeq (${OS}, linux)
	SED_INPLACE = sed -i
endif

.PHONY: all codegen devenv docker deploy clean mock pubsub

all: ${SERVICE_NAME}_${OS}

codegen:
	@#Generate MessagePacks. Add other directories containing definitions here.
	go generate ./api/middleware

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./${SERVICE_NAME} ./main.go
	docker build -t lisyaoran51/${SERVICE_NAME}:${GIT_COMMIT_HASH} . 
	docker tag lisyaoran51/${SERVICE_NAME}:${GIT_COMMIT_HASH} lisyaoran51/${SERVICE_NAME}
	docker push lisyaoran51/${SERVICE_NAME}

rollout: build
	kubectl set image deployment ${SERVICE_NAME}-deployment ${SERVICE_NAME}=lisyaoran51/${SERVICE_NAME}:${GIT_COMMIT_HASH} --record

proto:
	go get -u github.com/paper-trade-chatbot/be-proto

common:
	go get -u github.com/paper-trade-chatbot/be-common

pubsub:
	go get -u github.com/paper-trade-chatbot/be-pubsub