#!/bin/bash
# Will exit if there is an error
set -e -x

# First we want to run our tests
echo "Testing..."
go test -v ./...

# Builds the server
echo "Building server..."
go build -o bin/out
echo "Built server!"

# Run the server
echo "Running server!"
./bin/out
echo "Ran server..."
