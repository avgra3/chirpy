#!/bin/bash

set -e -x

# Needed environment variables
export GOOSE_DRIVER="postgres"
export GOOSE_DBSTRING="postgres://postgres:postgres@localhost:5432/chirpy"

echo "Running goose down migrations..."
goose -dir ./sql/schema down

echo "Running goose up migrations..."
goose -dir ./sql/schema up
