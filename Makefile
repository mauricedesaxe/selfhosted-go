# the default ARCH for tailwindcss is macos-x64
ARCH ?= macos-arm64

# Setup the project for the first time
setup:
	@echo "Setting up project..."

	@echo "Creating database directory..."
	@mkdir -p db
	@echo "Database directory created."

	@echo "Install templ"
	@go install github.com/a-h/templ/cmd/templ@latest
	@echo "templ installed."

	@echo "Downloading Tailwind CSS ($(ARCH))..."
	@curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-$(ARCH)
	@echo "Tailwind CSS ($(ARCH)) downloaded."

	@echo "Setting permissions..."
	@chmod +x tailwindcss-$(ARCH)
	@mv tailwindcss-$(ARCH) tailwindcss

	@echo "Initializing Tailwind CSS..."
	@./tailwindcss init
	@echo "Tailwind CSS initialized."

ENV ?= dev

tailwind:
ifeq ($(ENV), dev)
	@./tailwindcss -i ./css/main.css -o ./public/css/tailwind.css
else
	@./tailwindcss -i ./css/main.css -o ./public/css/tailwind.css --minify
endif

kill:
	@echo "Killing process on port 3000..."
	@kill -9 $(lsof -i:3000 -t) || echo "No process running on port 3000."

dev:
	@echo "Starting development server..."
	@air

build:
	@echo "Building project..."

	@echo "Building templates..."
	@templ generate || echo "Failed to build the templates."
	@echo "Templates built."

	@echo "Building Tailwind CSS..."
	@make tailwind ENV=prod || echo "Failed to build the Tailwind CSS."
	@echo "Tailwind CSS built."

	@echo "Building Go binary..."
	@go build -o bin/app || echo "Failed to build the Go binary."
	@chmod +x ./bin/app
	@echo "Go binary built."

	@echo "Project built."

run:
	@echo "Running project..."
	@./bin/app || echo "Failed to run the application. Check if the binary exists and has execution permissions."

clean:
	@echo "Deleting all contents of the /db directory..."
	@rm -rf db
	@mkdir db
	@echo "Database directory cleaned."