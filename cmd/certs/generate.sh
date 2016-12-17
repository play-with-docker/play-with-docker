#!/bin/bash

go run main.go

tar -cvf keys.tar ca.pem key.pem cert.pem
