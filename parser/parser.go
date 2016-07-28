package parser

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rodrigo-kayala/ahgora-cmd/reader"
)

const ahgoraLogin = "https://www.ahgora.com.br/externo/login"
const batidasURL = "https://www.ahgora.com.br/externo/batidas"

// UserRecord summary
type UserRecord struct {
	HoursBalance         time.Duration
	ClosingDate          time.Time
	StandardWorkingHours int
	HolydaysTilClosing   int
	TodayRecords         []time.Time
}

// WorkingDaysTilClosing returns the number of working days until closing
func (ur UserRecord) WorkingDaysTilClosing() int {
	return ur.getWorkDays() - ur.HolydaysTilClosing
}

// WorkingHoursTilClosing returns remaining workings hours until closing
func (ur UserRecord) WorkingHoursTilClosing() time.Duration {
	return time.Duration(ur.WorkingDaysTilClosing()*ur.StandardWorkingHours) * time.Hour
}

// MinutesAdjustmentPerDay returns total adjustment per day to clear hour bank at the closing
func (ur UserRecord) MinutesAdjustmentPerDay() time.Duration {
	minutesPerDay := math.Ceil((ur.HoursBalance.Minutes() * -1) / float64(ur.WorkingDaysTilClosing()))
	return time.Duration(minutesPerDay) * time.Minute
}

// WorkingHoursPerDayGoal returns total working hours per day goal to clean hour bank
func (ur UserRecord) WorkingHoursPerDayGoal() time.Duration {
	return time.Duration(float64(ur.StandardWorkingHours)+ur.MinutesAdjustmentPerDay().Hours()) * time.Hour
}

// TodayWorkedHours today worked hours
func (ur UserRecord) TodayWorkedHours() time.Duration {
	batidasArrLen := len(ur.TodayRecords)
	var durations []time.Duration

	for idx, batida := range ur.TodayRecords {
		if idx%2 != 0 {
			durations = append(durations, batida.Sub(ur.TodayRecords[idx-1]))
		} else if idx == batidasArrLen-1 {
			durations = append(durations, time.Now().Sub(batida))
		}
	}

	totalDuration := time.Duration(0)
	for _, duration := range durations {
		totalDuration = totalDuration + duration
	}

	return totalDuration
}

// LeaveAt returns da time de user should leave
func (ur UserRecord) LeaveAt() time.Time {
	pending := (time.Duration(8) * time.Hour) - ur.TodayWorkedHours()
	return time.Now().Add(pending + ur.MinutesAdjustmentPerDay())
}

func (ur UserRecord) String() string {
	return fmt.Sprintf(`Saldo: %v
Dias úteis até o fechamento: %v
Horas até o fechamento: %v
Minutos adicionais por dia: %v
Total desejado de trabalho por dia: %v
Batidas: %v
Horas trabalhadas: %v
Sair às: %v
`,
		ur.HoursBalance,
		ur.WorkingDaysTilClosing(),
		ur.WorkingHoursTilClosing(),
		ur.MinutesAdjustmentPerDay(),
		ur.WorkingHoursPerDayGoal(),
		ur.TodayRecords,
		ur.TodayWorkedHours(),
		ur.LeaveAt())
}

func (ur UserRecord) getWorkDays() int {
	today := zeroHourDate(time.Now())
	closing := zeroHourDate(ur.ClosingDate)

	days := 0
	for {
		if today.Equal(closing) {
			return days
		}
		if today.Weekday() != 6 && today.Weekday() != 7 {
			days++
		}
		today = today.Add(time.Hour * 24)
	}
}

// UserLogin type used for login in ahgora system
type UserLogin struct {
	Company  string
	Username string
	Password string
}

// Login returns sessionID of logged user
func (ul UserLogin) Login() (string, error) {
	values := url.Values{"empresa": {ul.Company}, "matricula": {ul.Username}, "senha": {ul.Password}}
	res, err := http.PostForm(ahgoraLogin, values)
	defer res.Body.Close()

	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		return "", fmt.Errorf("Response status not 200 - %s - code=%d", res.Status, res.StatusCode)
	}

	text, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return "", err
	}

	if string(text) != "{\"r\":\"success\"}" {
		return "", fmt.Errorf("Login not succesful = %s", text)
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

// NewUserRecord creates new UserRecord
func NewUserRecord(login UserLogin, closingDate time.Time, holydays int, workingHours int) UserRecord {
	sessionID, _ := login.Login()
	headers := map[string]string{"User-Agent": "ahgora-cmd"}
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

	ur := UserRecord{
		ClosingDate:          closingDate,
		HolydaysTilClosing:   holydays,
		StandardWorkingHours: workingHours,
		HoursBalance:         extractHoursBalance(res),
		TodayRecords:         extractRecords(res)}

	return ur
}

func extractHoursBalance(res *http.Response) time.Duration {
	start := "<td>SALDO</td>\n                            <td class=\"text-right danger\">\n  "
	end := "</td>\n                        </tr>"

	body, err := getHTMLPart(res.Body, start, end)
	if err != nil {
		panic(err)
	}

	balanceText := strings.TrimSpace(string(body))
	balance, _ := time.ParseDuration(strings.Replace(balanceText, ":", "h", 1) + "m")

	return balance
}

func extractRecords(res *http.Response) []time.Time {
	locSP, _ := time.LoadLocation("America/Sao_Paulo")

	todayStr := time.Now().Format("02/01/2006")
	// TODO: Melhorar isso
	start := todayStr + "                        \u003ctd rowspan=\"\"\u003e\n " +
		"                                                       08:00 as 17:00 - 08:00 as 17:00  " +
		"                      \u003c/td\u003e\n                        \u003ctd rowspan=\"\"\u003e"
	end := "\u003c/td\u003e\n                        \u003ctd\u003e"
	body, err := getHTMLPart(res.Body, start, end)

	if err != nil {
		panic(err)
	}

	text := strings.TrimSpace(string(body))

	batidasArr := strings.Split(text, ",")

	var records []time.Time

	for _, batida := range batidasArr {
		dateStr := fmt.Sprintf("%s %s", todayStr, strings.TrimSpace(batida))
		record, _ := time.ParseInLocation("02/01/2006 15:04", dateStr, locSP)

		records = append(records, record)
	}

	return records
}

func getHTMLPart(r io.Reader, startMark string, endMark string) (string, error) {
	str := reader.NewSkipTillReader(r, []byte(startMark))
	rtr := reader.NewReadTillReader(str, []byte(endMark))
	bs, err := ioutil.ReadAll(rtr)
	if err != nil {
		return "", err
	}
	text := string(bs)
	text = strings.Replace(text, startMark, "", 1)
	text = strings.Replace(text, endMark, "", 1)
	return text, err
}

func zeroHourDate(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 24, 0, 0, 0, time.UTC)
}
