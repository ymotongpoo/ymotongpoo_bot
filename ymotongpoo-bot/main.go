package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const CommandPrefix = "$$"

type Status struct {
	Events []Event `json:"events"`
}

type Event struct {
	Id      int      `json:"event_id"`
	Message *Message `json:"message"`
}

type Message struct {
	Id              string `json:"id"`
	Room            string `json:"room"`
	PublicSessionId string `json:"public_session_id"`
	IconUrl         string `json:"icon_url"`
	Type            string `json:"type"`
	SpeakerId       string `json:"speaker_id"`
	Nickname        string `json:"nickname"`
	Text            string `json:"text"`
}

type Finance struct {
	Results RateResults `json:"results"`
}

type RateResults struct {
	Rates []Rate `json:"rate"`
}

// Rate struct for Yahoo Finance
type Rate struct {
	Id   string  `json:"id"`
	Name string  `json:"Name"`
	Rate float64 `json:"Rate"`
	Date string  `json:"Date"`
	Time string  `json:"Time"`
	Ask  float64 `json:"Ask"`
	Bid  float64 `json:"Bid"`
}

func (r Rate) String() string {
	tokens := strings.Split(r.Name, " ")
	if len(tokens) == 3 {
		return fmt.Sprintf("1%v = %vJPY [%v %v]",
			tokens[0],
			r.Rate,
			r.Date,
			r.Time,
		)
	}
	return ""
}

var CommandMap = make(map[string](func(args []string) string))

const (
	YahooFinanceAPI = "http://query.yahooapis.com/v1/public/yql"
)

func FetchGooglePlusPost(id, lastPost string) {
	return
}

// JPY returns exchange rate for each unit foreign currencies.
func JPY(args []string) string {
	q := `select * from yahoo.finance.xchange where pair in ` +
		`("USDJPY","EURJPY","GBPJPY","CNYJPY")`
	v := url.Values{}
	v.Set("q", q)
	v.Set("format", "json")
	v.Set("diagonstics", "true")
	v.Set("env", "store://datatables.org/alltableswithkeys")
	urlStr := YahooFinanceAPI + "?" + v.Encode()

	resp, err := http.Get(urlStr)
	if err != nil {
		log.Println(err.Error())
		return ""
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	log.Println(string(data))

	var f Finance
	err = json.NewDecoder(resp.Body).Decode(&f)
	if err != nil {
		log.Println(err.Error())
		return ""
	}

	results := make([]string, len(f.Results.Rates))
	for i, r := range f.Results.Rates {
		results[i] = r.String()
	}

	return strings.Join(results, " / ")
}

func main() {
	CommandMap["jpy"] = JPY

	// Start polling
	go func() {
		for {
			select {
			case <-time.After(5 * time.Minute):
				//go FetchGooglePlusPost(_, _)
			}
		}
	}()

	// Start serving
	http.HandleFunc("/", handler)
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		panic(err)
	}
}

// handler routes HTTP request from Lingr based on HTTP method.
func handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var status Status
		err := json.NewDecoder(r.Body).Decode(&status)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
		}
		results := handleEvents(status.Events)
		if len(results) > 0 {
			results = strings.TrimRight(results, "\n ")
			if runes := []rune(results); len(runes) > 1000 {
				results = string(runes[0:999])
			}
			fmt.Fprintln(w, results)
		}

	case "GET":
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err.Error())
		}
		log.Printf("Get request: %v", string(data))
		fmt.Fprintln(w, "とんぷbotです")
	}
}

// execCommand returns command result string based on event message string.
func execCommand(ev Event) (result string) {
	tokens := strings.Split(ev.Message.Text, " ")
	if strings.HasPrefix(tokens[0], CommandPrefix) {
		commandStr := strings.TrimLeft(tokens[0], CommandPrefix)
		command, exist := CommandMap[commandStr]
		if exist {
			return command(tokens[1:])
		} else {
			return fmt.Sprintf("しらないコマンド: %s\n", tokens[0])
		}
	}
	return result
}

func handleEvents(events []Event) (results string) {
	for _, ev := range events {
		results += execCommand(ev)
	}
	return results
}
