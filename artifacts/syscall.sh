#!/usr/bin/env sh
# Try to make a forbidden syscall that should be blocked by seccomp
echo "Testing syscall blocking..."
# This should be blocked by seccomp
exec 3<>/dev/tcp/1.1.1.1/53 2>/dev/null || echo "syscall blocked (expected)"
