package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rodrigo-kayala/ahgora-cmd/parser"
)

const ahgoraLogin = "https://www.ahgora.com.br/externo/login"
const batidasURL = "https://www.ahgora.com.br/externo/batidas"
const ahgoraCompanyEnv = "AHGORA_COMPANY"
const ahgoraUsernameEnv = "AHGORA_USERNAME"
const ahgoraPasswordEnv = "AHGORA_PASSWORD"

func main() {
	flag.String("help", "", "display this help")
	flag.Parse()

	company := os.Getenv(ahgoraCompanyEnv)
	if company == "" {
		panic("company env " + ahgoraCompanyEnv + " not set")
	}

	username := os.Getenv(ahgoraUsernameEnv)
	if username == "" {
		panic("username env " + ahgoraUsernameEnv + " not set")
	}

	password := os.Getenv(ahgoraPasswordEnv)
	if password == "" {
		panic("password env " + ahgoraPasswordEnv + " not set")
	}

	closingDate := time.Date(2016, time.October, 31, 24, 0, 0, 0, time.UTC)

	login := parser.UserLogin{
		Company:  company,
		Username: username,
		Password: password}
	record := parser.NewUserRecord(login, closingDate, 0, 8)
	fmt.Printf(record.String())
}
