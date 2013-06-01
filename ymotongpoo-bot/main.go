package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// CommandPrefix defines prefix characters to call Lingr bot from Lingr.
const CommandPrefix = "$$"

// Structs for Lingr bot push notificaion.
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

// CommandMap holds pairs of command name and counterpart function name.
type CommandMap map[string](func(args []string) string)

var commandMap CommandMap

const (
	YahooFinanceAPI = "http://query.yahooapis.com/v1/public/yql"
	RedditAPI       = "http://www.reddit.com/r/golang/hot.json"
)

// Structs for Yahoo Finance
type Finance struct {
	Query struct {
		Results struct {
			Rates []Rate `json:"rate"`
		} `json:"results"`
	} `json:"query"`
}

type Rate struct {
	Id   string `json:"id"`
	Name string `json:"Name"`
	Rate string `json:"Rate"`
	Date string `json:"Date"`
	Time string `json:"Time"`
	Ask  string `json:"Ask"`
	Bid  string `json:"Bid"`
}

// String returns formatted string of a currency rate.
func (r Rate) String() string {
	tokens := strings.Split(r.Name, " ")
	if len(tokens) == 3 {
		return fmt.Sprintf("1%v = %vJPY [%v %v]",
			tokens[0],
			r.Rate,
			r.Date,
			r.Time)
	}
	return ""
}

// Structs for Reddit
type Reddit struct {
	Data struct {
		Children []struct {
			ItemData ItemData `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type ItemData struct {
	Score int    `json:"score"`
	Ups   int    `json:"ups"`
	Downs int    `json:"downs"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// String returns formatted string of a reddit item.
func (i ItemData) String() string {
	return fmt.Sprintf("%v (u:%v, d:%v) %v : %v",
		i.Score,
		i.Ups,
		i.Downs,
		i.Title,
		i.URL)
}

// Help returns a command list.
func Help(args []string) string {
	usage := "とんぷbot command list\n"
	for k, _ := range commandMap {
		usage += k + "\n"
	}
	return usage
}

// JPY returns exchange rate for each unit foreign currencies.
func JPY(args []string) string {
	q := `select * from yahoo.finance.xchange where pair in ` +
		`("USDJPY","EURJPY","GBPJPY","CADJPY","CNYJPY")`
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

	var f Finance
	err = json.NewDecoder(resp.Body).Decode(&f)
	if err != nil {
		log.Println(err.Error())
		return ""
	}

	results := make([]string, len(f.Query.Results.Rates))
	for i, r := range f.Query.Results.Rates {
		results[i] = r.String()
	}

	return strings.Join(results, " / ")
}

// RedditHot returns hot topic in Reddit under subreddit "golang".
// This takes 1st argument as upper limit of fetching item.
// Default is 3 and max is 5.
func RedditHot(args []string) string {
	v := url.Values{}
	if len(args) > 0 {
		limit, err := strconv.Atoi(args[0])
		if err != nil {
			return "引数が変だよ: " + strings.Join(args, " ")
		}
		if limit > 5 {
			v.Set("limit", "5")
		} else {
			v.Set("limit", args[0])
		}
	} else {
		v.Set("limit", "3")
	}
	urlStr := RedditAPI + "?" + v.Encode()
	log.Println(urlStr)

	resp, err := http.Get(urlStr)
	if err != nil {
		log.Println(err.Error())
		return ""
	}
	defer resp.Body.Close()

	var r Reddit
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		log.Println(err.Error())
		return ""
	}

	results := make([]string, len(r.Data.Children.ItemData))
	for i, d := range r.Data.Children.ItemData {
		results[i] = d.String()
	}
	return strings.Join(results, "\n")
}

func init() {
	commandMap = CommandMap{
		"help":   Help,
		"jpy":    JPY,
		"reddit": RedditHot,
	}
}

// main
func main() {
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
		command, exist := commandMap[commandStr]
		if exist {
			return command(tokens[1:])
		} else {
			return fmt.Sprintf("しらないコマンド: %s\n", tokens[0])
		}
	}
	return result
}

// handleEvents returns
func handleEvents(events []Event) (results string) {
	for _, ev := range events {
		results += execCommand(ev)
	}
	return results
}
