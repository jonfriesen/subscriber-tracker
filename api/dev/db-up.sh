#!/bin/bash

docker run --rm -p 5432:5432 --name dev-postgres -e POSTGRES_PASSWORD=mysecretpassword -d postgres:12.3-alpine