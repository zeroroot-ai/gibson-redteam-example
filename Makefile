# Root Makefile — github.com/zeroroot-ai/gibson-redteam-example
#
# Reference red-team trio. Each component is its own Go module with its own
# Makefile (build / test / register / run / image); the `missions` directory
# holds CUE mission definitions with no Go build. This root Makefile aggregates
# the components to satisfy the org Makefile contract (build / test / check);
# see .github#170.

COMPONENTS := findings-sink llm-redteam prompt-probe

.PHONY: build test check

build: ## Build every component.
	@set -e; for c in $(COMPONENTS); do echo "==> build $$c"; $(MAKE) -C $$c build; done

test: ## Test every component.
	@set -e; for c in $(COMPONENTS); do echo "==> test $$c"; $(MAKE) -C $$c test; done

check: ## Static-check gate: go fmt + go vet across every component.
	@set -e; for c in $(COMPONENTS); do \
		echo "==> check $$c"; \
		( cd $$c && go fmt ./... && go vet ./... ); \
	done; \
	echo "All checks passed!"
