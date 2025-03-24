#!/bin/bash

# Builds the server
echo "Building server..."
go build -o bin/out
echo "Built server!"

# Run the server
echo "Running server!"
./bin/out
echo "Ran server..."
