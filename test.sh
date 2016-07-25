#!/bin/bash

req() {
	user="gurbieta"
	if [ "$2" != "" ]; then
		user=$2
	fi
	curl -s "localhost:8080" -d "token=xRPuTAhD7scFGno6zbdcnwff" -d "text=$1" -d "user_name=$user";
}

assert() {
	if [ "$1" != "$2" ]; then
		echo "Expected $1 but get $2"
	else
		echo "OK - $1"
	fi
}

resp=$(req "bot batidas")
assert '{"text":"Usuário não cadastrado - cadastre-se =\u003e bot reg user:pass"}' "$resp"

resp=$(req "bot reg 12662:1234")
assert '{"text":"Usuário cadastrado\nMatrícula: 12662\nSenha: 1234"}' "$resp"

resp=$(req "bot reg 12662:1234")
assert '{"text":"Usuário já cadastrado"}' "$resp"

resp=$(req "bot batidas")
assert '{"text":"Usuário já cadastrado"}' "$resp"
