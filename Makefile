default: build

.PHONY: frontend
frontend:
	cd frontend && npm ci
	cd frontend && npm run build


.PHONY: build
build: frontend
	go build

.PHONY: install
install: frontend
	go install
