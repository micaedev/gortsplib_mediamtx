BASE_IMAGE = golang:1.23-alpine3.20
LINT_IMAGE = golangci/golangci-lint:v1.64.5

.PHONY: $(shell ls)

help:
	@echo "usage: make [action]"
	@echo ""
	@echo "available actions:"
	@echo ""
	@echo "  mod-tidy        run go mod tidy"
	@echo "  format          format source files"
	@echo "  test            run tests"
	@echo "  test32          run tests on a 32-bit system"
	@echo "  test-highlevel  run high-level tests"
	@echo "  lint            run linter"
	@echo "  bench           run benchmarks"
	@echo ""

blank :=
define NL

$(blank)
endef

include scripts/*.mk
