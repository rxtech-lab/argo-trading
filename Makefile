.PHONY: generate clean

# Generate Go code from proto files and run go generate
generate:
	cd pkg/strategy && protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative strategy.proto
	go generate ./...

# Clean generated files
clean:
	cd pkg/strategy && rm -f *.pb.go 