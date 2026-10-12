GOLIB ?= golib

.PHONY: source-docs-check check check-full ci cohesion docs inventory repository-check specification-check specification-check-online workflows

check: source-docs-check
	$(GOLIB) check --local --all

check-full: source-docs-check repository-check cohesion specification-check-online
	$(GOLIB) check --all

ci: source-docs-check repository-check cohesion specification-check check

cohesion:
	$(GOLIB) cohesion check

docs:
	./scripts/check-docs.sh

inventory repository-check:
	$(GOLIB) repository check

specification-check:
	$(GOLIB) specification check

specification-check-online:
	$(GOLIB) specification check --online

workflows:
	$(GOLIB) workflows check

source-docs-check:
	$(GOLIB) source-docs check --json
