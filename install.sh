#!/usr/bin/env bash
go build -o aic main.go
mkdir -p ~/.local/bin
mv aic ~/.local/bin/
echo "aic installed successfully in ~/.local/bin/"
