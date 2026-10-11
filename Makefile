GOLIB ?= golib

.PHONY: check check-full ci cohesion docs inventory repository-check specification-check specification-check-online workflows

check:
	$(GOLIB) check --local --all

check-full: repository-check cohesion specification-check-online
	$(GOLIB) check --all

ci: repository-check cohesion specification-check check

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
