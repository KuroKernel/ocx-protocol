#!/usr/bin/env sh
nc -z -w1 1.1.1.1 53 >/dev/null 2>&1 || echo "net blocked (expected)"
