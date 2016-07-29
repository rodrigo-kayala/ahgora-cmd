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

	help := flag.Bool("h", false, "display this help")
	cd := flag.String("c", "2016-11-01", "closing date in format YYYY-MM-DD")
	holidays := flag.Int("ho", 0, "holydays until closing")
	workingHours := flag.Int("w", 8, "working hours per day")
	flag.Parse()

	if *help == true {
		flag.PrintDefaults()
		return
	}

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

	closingDate, err := time.Parse("2006-01-02", *cd)
	if err != nil {
		panic(err)
	}

	fmt.Printf("params: -c %v -ho %v -w %v\n", closingDate, *holidays, *workingHours)

	login := parser.UserLogin{
		Company:  company,
		Username: username,
		Password: password}
	record := parser.NewUserRecord(login, closingDate, *holidays, *workingHours)
	fmt.Printf(record.String())
}
