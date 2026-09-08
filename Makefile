GOLIB ?= golib

.PHONY: check ci cohesion docs inventory repository-check specification-check workflows

check:
	$(GOLIB) check --all

ci: repository-check cohesion specification-check check

cohesion:
	$(GOLIB) cohesion check

docs:
	./scripts/check-docs.sh

inventory repository-check:
	$(GOLIB) repository check

specification-check:
	$(GOLIB) specification check --online

workflows:
	$(GOLIB) workflows check
