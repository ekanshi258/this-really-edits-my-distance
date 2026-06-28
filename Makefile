PORT ?= 8080
 
.PHONY: build dev
 
# Generate public/graph.json
build:
	cd build_libs && go run .
 
# Build then serve public/ on localhost
dev: build
	cd public && python3 -m http.server $(PORT)