default: build

.PHONY: watch
watch:
	npm --prefix extension run watch

.PHONY: install
install: frontend
	go install
