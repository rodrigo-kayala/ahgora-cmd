#!/bin/bash
OS_NAME="$(uname)"

case "$OS_NAME" in
  ("Linux") sudo curl -Lo /usr/local/bin/ahgora https://github.com/rodrigo-kayala/ahgora-cmd/releases/download/1.0.0/ahgora-cmd-linux-x64 ;;
  ("Darwin") sudo curl -Lo /usr/local/bin/ahgora https://github.com/rodrigo-kayala/ahgora-cmd/releases/download/1.0.0/ahgora-cmd-mac-x64 ;;
  (*) exit 2 ;;
esac

sudo chmod +x /usr/local/bin/ahgora


