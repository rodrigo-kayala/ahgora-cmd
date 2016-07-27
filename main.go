package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
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

	sessionID, err := getSessionID(company, username, password)
	if err != nil {
		panic(err)
	}

	printBatidas(sessionID)
}

func makeRequest(url string, method string, headers map[string]string, cookies map[string]string) (*http.Response, error) {

	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	for key, value := range cookies {
		cookie := http.Cookie{Name: key, Value: value}
		req.AddCookie(&cookie)
	}

	client := http.Client{}
	return client.Do(req)

}

func getWorkDays() int {
	t := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 24, 0, 0, 0, time.UTC)
	f := time.Date(2016, time.October, 31, 24, 0, 0, 0, time.UTC)
	days := 0
	for {
		if t.Equal(f) {
			return days
		}
		if t.Weekday() != 6 && t.Weekday() != 7 {
			days++
		}
		t = t.Add(time.Hour * 24)
	}
}

func printBatidas(sessionID string) {
	headers := map[string]string{"User-Agent": "gurbieta-bot"}
	cookies := map[string]string{"PHPSESSID": sessionID}

	res, err := makeRequest(batidasURL, "GET", headers, cookies)
	defer res.Body.Close()

	if err != nil {
		panic(err)
	}

	if res.StatusCode != 200 {
		panic("Response status not 200 - " + string(res.StatusCode))
	}

	defer res.Body.Close()

	locSP, _ := time.LoadLocation("America/Sao_Paulo")

	now := time.Now().In(locSP)

	start := "<td>SALDO</td>\n                            <td class=\"text-right danger\">\n  "
	end := "</td>\n                        </tr>"

	body, _, err := getHTMLPart(res.Body, start, end)

	if err != nil {
		panic(err)
	}

	saldo := strings.TrimSpace(string(body))
	saldoDuration, _ := time.ParseDuration(strings.Replace(saldo, ":", "h", 1) + "m")
	fmt.Print("Saldo: " + saldoDuration.String() + "\n")

	workHours := getWorkDays() * 8

	fmt.Printf("Dias uteis até o fechamento: %d\n", workHours/8)
	fmt.Printf("Horas até o fechamento: %d\n", workHours)
	minutesPerDay := math.Ceil(((saldoDuration.Minutes() * -1) / float64(getWorkDays())))
	fmt.Printf("Minutos adicionais por dia: %s\n", (time.Duration(minutesPerDay) * time.Minute).String())
	fmt.Printf("Total desejado de trabalho por dia: %s\n", time.Duration(minutesPerDay+(8*60))*time.Minute)

	todayStr := now.Format("02/01/2006")
	start = todayStr + "                        \u003ctd rowspan=\"\"\u003e\n                                                        08:00 as 17:00 - 08:00 as 17:00                        \u003c/td\u003e\n                        \u003ctd rowspan=\"\"\u003e"
	end = "\u003c/td\u003e\n                        \u003ctd\u003e"
	body, _, err = getHTMLPart(res.Body, start, end)

	if err != nil {
		panic(err)
	}

	text := string(body)

	if text == "" {
		panic("cannot get time")
	}

	batidasArr := strings.Split(text, ",")
	batidasArrLen := len(batidasArr)

	if batidasArrLen == 0 {
		text = "Ainda não tem batidas para o " + todayStr
	} else {
		text = "Batidas - " + text + "\n"

		var durations []time.Duration

		for idx, batida := range batidasArr {
			if idx%2 != 0 {
				dateStr1 := todayStr + " " + strings.Trim(batida, " ")
				dateStr2 := todayStr + " " + strings.Trim(batidasArr[idx-1], " ")
				t1, _ := time.ParseInLocation("02/01/2006 15:04", dateStr1, locSP)
				t2, _ := time.ParseInLocation("02/01/2006 15:04", dateStr2, locSP)

				durations = append(durations, t1.Sub(t2))
			} else if idx == batidasArrLen-1 {
				dateStr := todayStr + " " + strings.Trim(batida, " ")
				t1, _ := time.ParseInLocation("02/01/2006 15:04", dateStr, locSP)

				durations = append(durations, now.Sub(t1))
			}
		}

		totalDuration := time.Duration(0)
		for _, duration := range durations {
			totalDuration = totalDuration + duration
		}

		text += "Horas trabalhadas - " + (totalDuration - (totalDuration % time.Second)).String() + "\n"

		pending := (time.Duration(8) * time.Hour) - totalDuration

		text += "Restante - " + ((pending - (pending % time.Second)) + time.Duration(minutesPerDay)*time.Minute).String() + "\n"

		if batidasArrLen%2 != 0 {
			text += "Sair as - " + now.Add(pending+time.Duration(minutesPerDay)*time.Minute).Format("15:04") + "\n"
		}
	}

	fmt.Print(text)
}

func getSessionID(company string, username string, password string) (string, error) {
	values := url.Values{"empresa": {company}, "matricula": {username}, "senha": {password}}
	res, err := http.PostForm(ahgoraLogin, values)
	defer res.Body.Close()

	if err != nil {
		panic(err)
	}

	if res.StatusCode != 200 {
		panic("Response status not 200 - " + res.Status + " - code=" + string(res.StatusCode))
	}

	text, err := ioutil.ReadAll(res.Body)

	if err != nil {
		panic(err)
	}

	if string(text) != "{\"r\":\"success\"}" {
		panic("Login not succesful = " + string(text))
	}

	sessionID := ""

	for _, cookie := range res.Cookies() {
		if cookie.Name == "PHPSESSID" {
			sessionID = cookie.Value
			break
		}
	}

	if sessionID == "" {
		return "", errors.New("cannot retrieve sessionId")
	}

	return sessionID, nil
}

func getHTMLPart(reader io.Reader, startMark string, endMark string) (string, *SkipTillReader, error) {
	str := NewSkipTillReader(reader, []byte(startMark))
	rtr := NewReadTillReader(str, []byte(endMark))
	bs, err := ioutil.ReadAll(rtr)
	if err != nil {
		return "", nil, err
	}
	text := string(bs)
	text = strings.Replace(text, startMark, "", 1)
	text = strings.Replace(text, endMark, "", 1)
	return text, str, err
}
