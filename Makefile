.PHONY: help dev build build-frontend build-backend run clean install-deps

help:
	@echo "WebSSH Makefile Commands:"
	@echo "  make install-deps    - Install Go and Node dependencies"
	@echo "  make dev             - Run development server (backend + frontend)"
	@echo "  make build           - Build production binary with embedded frontend"
	@echo "  make build-frontend  - Build frontend only"
	@echo "  make build-backend   - Build backend only"
	@echo "  make run             - Run the application"
	@echo "  make clean           - Clean build artifacts"

install-deps:
	@echo "Installing Go dependencies..."
	go mod download
	@echo "Installing frontend dependencies..."
	cd web && npm install

dev:
	@echo "Starting development server..."
	@echo "Backend will run on http://localhost:8080"
	@echo "Frontend dev server will run on http://localhost:5173"
	@echo ""
	@echo "Make sure to set environment variables:"
	@echo "  WEBSSH_JWT_SECRET=your-secret-key"
	@echo "  WEBSSH_ENCRYPTION_KEY=your-32-byte-encryption-key"
	go run cmd/server/main.go

build-frontend:
	@echo "Building frontend..."
	cd web && npm run build

build-backend:
	@echo "Building backend..."
	go build -o bin/webssh.exe cmd/server/main.go

build: build-frontend build-backend
	@echo "Build complete! Binary: bin/webssh.exe"

run:
	@echo "Running WebSSH..."
	./bin/webssh.exe

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf web/dist/
	rm -rf data/
	rm -rf logs/
