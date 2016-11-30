#!/bin/bash

docker build -t builder .&& docker run --rm builder | docker build -t franela/play-with-docker:latest -
