package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/urlfetch"
)

var app_token = "xRPuTAhD7scFGno6zbdcnwff"
var trigger = "bot "

var sessions = map[string]string{}

type SlackRequest struct {
	Token        string
	TeamId       int
	Team_domain  string
	Channel_id   int
	Channel_name string
	User_id      int
	User_name    string
	Text         string
	Trigger_word string
}

type AhgoraUser struct {
	SlackUser string
	Company   string
	Login     string
	Password  string
}

func init() {
	http.HandleFunc("/", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {
	slackRequest := SlackRequest{
		Token:     r.PostFormValue("token"),
		User_name: r.PostFormValue("user_name"),
		Text:      r.PostFormValue("text"),
	}

	token := slackRequest.Token

	if token != app_token {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if "slackbot" == slackRequest.User_name {
		http.Error(w, "Slackbot user", http.StatusOK)
		return
	}

	command := strings.Trim(slackRequest.Text, " ")

	if len(command) < len(trigger)+1 {
		http.Error(w, "Text param must be in the form '"+trigger+" command'", http.StatusBadRequest)
		return
	}

	command = strings.TrimPrefix(command, trigger)

	command = strings.Trim(command, " ")

	if command == "batidas" {
		ahgoraBatidas(slackRequest.User_name, w, r)
	} else if strings.Index(command, "reg ") == 0 {
		register(slackRequest.User_name, command, w, r)
	} else {
		respond(w, "No command")
	}
}

func getHtmlPart(reader io.Reader, startMark string, endMark string) (string, *SkipTillReader, error) {
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

func ahgoraLogin(user AhgoraUser, r *http.Request) string {
	ctx := appengine.NewContext(r)
	client := urlfetch.Client(ctx)

	values := url.Values{"empresa": {user.Company}, "matricula": {user.Login}, "senha": {user.Password}}
	res, err := client.PostForm("https://www.ahgora.com.br/externo/login", values)

	if err != nil {
		panic(err)
	}

	if res.StatusCode != 200 {
		panic("Response status not 200 - " + res.Status + " - code=" + string(res.StatusCode))
	}

	defer res.Body.Close()

	text, err := ioutil.ReadAll(res.Body)

	if err != nil {
		panic(err)
	}

	if (string(text) != "{\"r\":\"success\"}") {
		panic("Login not succesful = " + string(text))
	}

	sessionId := ""

	for _, cookie := range res.Cookies() {
		if cookie.Name == "PHPSESSID" {
			sessionId = cookie.Value
			break
		}
	}

	sessions[user.SlackUser] = sessionId

	return sessionId
}

func register(user string, command string, w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	k := datastore.NewKey(c, "AhgoraUser", user, 0, ahgoraUserKey(c))

	var ahgoraUser AhgoraUser

	err := datastore.Get(c, k, &ahgoraUser)

	if (err != nil) {
		if (err == datastore.ErrNoSuchEntity) {
			userAndPassArr := strings.Split(strings.TrimPrefix(command, "reg "), ":")

			if (len(userAndPassArr) != 2) {
				panic("User and pass format must be user:pass")
			}

			ahgoraUser = AhgoraUser{
				SlackUser: user,
				Company:   "a382748",
				Login:     userAndPassArr[0],
				Password:  userAndPassArr[1],
			}

			t, err := datastore.Put(c, k, &ahgoraUser)
			
			fmt.Println(t)

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			respond(w, "Usuário cadastrado\nMatrícula: " + ahgoraUser.Login + "\nSenha: " + ahgoraUser.Password)
		} else {
			panic(err)
		}
	} else {
		respond(w, "Usuário já cadastrado")
	}
}

func ahgoraBatidas(user string, w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	k := datastore.NewKey(c, "AhgoraUser", user, 0, ahgoraUserKey(c))

	var ahgoraUser AhgoraUser

	err := datastore.Get(c, k, &ahgoraUser)

	if (err != nil) {
		if (err == datastore.ErrNoSuchEntity) {
			respond(w, "Usuário não cadastrado - cadastre-se => bot reg user:pass")
			return
		} else {
			panic(err)
		}
	}

	if sessions[user] == "" {
		ahgoraLogin(ahgoraUser, r)
	}

	batidasUrl := "https://www.ahgora.com.br/externo/batidas"

	headers := map[string]string{ "User-Agent": "gurbieta-bot", }
	cookies := map[string]string{ "PHPSESSID": sessions[user], }

	res, err := makeRequest(r, batidasUrl, "GET", headers, cookies)

	if err != nil {
		panic(err)
	}

	if res.StatusCode != 200 {
		panic("Response status not 200 - " + string(res.StatusCode))
	}

	defer res.Body.Close()

	locSP, _ := time.LoadLocation("America/Sao_Paulo")

	now := time.Now().In(locSP)

	todayStr := now.Format("02/01/2006")

	start := todayStr + "                        \u003ctd rowspan=\"\"\u003e\n                                                        09:00 as 18:00 - 09:00 as 18:00                        \u003c/td\u003e\n                        \u003ctd rowspan=\"\"\u003e"
	end := "\u003c/td\u003e\n                        \u003ctd\u003e"

	body, _, err := getHtmlPart(res.Body, start, end)

	if err != nil {
		panic(err)
	}

	text := string(body)

	// TODO: delete after test
	// text = "09:35, 12:18, 14:17"

	batidasArr := strings.Split(text, ",")
	batidasArrLen := len(batidasArr)

	if (batidasArrLen == 0) {
		text = "Ainda não tem batidas para o " + todayStr
	} else {
		text = "Batidas - " + text + "\n"

		var durations []time.Duration

		for idx, batida := range batidasArr {
			if idx % 2 != 0 {
				dateStr1 := todayStr + " " + strings.Trim(batida, " ")
				dateStr2 := todayStr + " " + strings.Trim(batidasArr[idx-1], " ")
				t1, _ := time.ParseInLocation("02/01/2006 15:04", dateStr1, locSP)
				t2, _ := time.ParseInLocation("02/01/2006 15:04", dateStr2, locSP)
				
				durations = append(durations, t1.Sub(t2))
			} else if idx == batidasArrLen - 1 {
				dateStr := todayStr + " " + strings.Trim(batida, " ")
				t1, _ := time.ParseInLocation("02/01/2006 15:04", dateStr, locSP)
				
				durations = append(durations, now.Sub(t1))
			}
		}
		
		totalDuration := time.Duration(0)
		for _, duration := range durations {
			totalDuration = totalDuration + duration
		}

		text += "Horas trabalhadas - " + totalDuration.String() + "\n"

		pending := (time.Duration(8) * time.Hour) - totalDuration

		text += "Restante - " + pending.String() + "\n"

		if (batidasArrLen % 2 != 0) {
			text += "Sair as - " + now.Add(pending).Format("15:04") + "\n"
		}
	}

	respond(w, text)
}

func makeRequest(r *http.Request, url string, method string, headers map[string]string, cookies map[string]string) (*http.Response, error) {
	ctx := appengine.NewContext(r)
	client := urlfetch.Client(ctx)

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

	return client.Do(req)
}

func respond(w http.ResponseWriter, text string) {
	m := make(map[string]interface{})
	m["text"] = text

	b, err := json.Marshal(m)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(b))
}

func ahgoraUserKey(c appengine.Context) *datastore.Key {
	return datastore.NewKey(c, "AhgoraUser", "default_ahgorauser", 0, nil)
}
