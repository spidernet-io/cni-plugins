include Makefile.defs

all: usage

usage:
	@echo "usage:"
	@echo  "  \033[35m make build \033[0m:       --- build all plugins"
	@echo  "  \033[35m make image \033[0m:       --- build docker image"
	@echo  "  \033[35m make e2e \033[0m:         --- NO IMPLEMENT! "

.PHONY: build

build:
	@mkdir -p ./.tmp/bin ; \
	for plugin in `ls ./plugins/` ; do   \
		echo "build $${plugin} to $(ROOT_DIR)/.tmp/bin/${plugin}" ; \
		$(GO_BUILD_FLAGS) $(GO_BUILD) -o ./.tmp/bin/$${plugin} ./plugins/$${plugin} ;  \
	done
