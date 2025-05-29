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

.PHONY: clean
clean:
	rm -rf frontend/node_modules
	rm -rf frontend/dist
	go clean

