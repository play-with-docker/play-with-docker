#!/bin/bash

docker build -t builder .&& docker run --rm builder | sudo docker build -t franela/play-with-docker:latest -
