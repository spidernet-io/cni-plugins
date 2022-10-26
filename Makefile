include Makefile.defs

all: usage

usage:
	@echo "usage:"
	@echo  "  \033[35m make build \033[0m:       --- build all plugins"
	@echo  "  \033[35m make image \033[0m:       --- build docker image"
	@echo  "  \033[35m make test \033[0m:        --- run e2e test on your local environment"

.PHONY: build
build:
	@mkdir -p ./.tmp/bin ; \
	for plugin in `ls ./plugins/` ; do   \
		echo "build $${plugin} to $(ROOT_DIR)/.tmp/bin/${plugin}" ; \
		$(GO_BUILD_FLAGS) $(GO_BUILD) -o ./.tmp/bin/$${plugin} ./plugins/$${plugin} ;  \
	done

.PHONY: lint-golang
lint-golang:
	GOOS=linux golangci-lint run ./...

.PHONY: lint_image_trivy
lint_image_trivy: IMAGE_NAME ?=
lint_image_trivy:
	[ -n "$(IMAGE_NAME)" ] || { echo "error, please input IMAGE_NAME" && exit 1 ; }  ; \
 	IS_TAR_FILE=` echo $(IMAGE_NAME) | egrep \.tar$ ` ; \
 	if [ -z "${IS_TAR_FILE}" ]; then  \
		docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
			  -v /tmp/trivy:/root/trivy.cache/  \
			  aquasec/trivy:latest image --exit-code 1  --severity $(LINT_TRIVY_SEVERITY_LEVEL)  ${IMAGE_NAME}  ;\
 	else  \
		docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
			  -v /tmp/trivy:/root/trivy.cache/ \
			  -v $(IMAGE_NAME):$(IMAGE_NAME)  \
			  aquasec/trivy:latest image --exit-code 1  --severity $(LINT_TRIVY_SEVERITY_LEVEL)  --input ${IMAGE_NAME}  ;\
 	fi

.PHONY: lint_chart_trivy
lint_chart_trivy:
	@ docker run --rm \
 		  -v /tmp/trivy:/root/trivy.cache/  \
          -v $(ROOT_DIR):/tmp/src  \
          aquasec/trivy:latest config --exit-code 1  --severity $(LINT_TRIVY_SEVERITY_LEVEL) /tmp/src/charts


.PHONY: lint_dockerfile_trivy
lint_dockerfile_trivy:
	@ docker run --rm \
 		  -v /tmp/trivy:/root/trivy.cache/  \
          -v $(ROOT_DIR):/tmp/src  \
          aquasec/trivy:latest config --exit-code 1  --severity $(LINT_TRIVY_SEVERITY_LEVEL) /tmp/src/images
