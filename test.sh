#!/bin/bash

req() {
	user="gurbieta"
	if [ "$2" != "" ]; then
		user=$2
	fi
	curl -s "localhost:8080" -d "token=hospNwvYl5EdtWuoZvHiawfr" -d "text=$1" -d "user_name=$user";
}

assert() {
	if [ "$1" != "$2" ]; then
		echo "Expected $1 but get $2"
	else
		echo "OK - $1"
	fi
}

resp=$(req "bot batidas")
assert '{"text":"batidas"}' "$resp"
