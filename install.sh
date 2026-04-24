#!/bin/bash
go build -o aic main.go
mkdir -p ~/.local/bin
mv aic ~/.local/bin/
echo "✅ aic успешно обновлен в ~/.local/bin/"
