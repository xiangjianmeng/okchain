#!/bin/bash -e

./checkcode.sh | grep /|xargs gofmt -l -s -w