package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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

var CommandMap = make(map[string](func(args []string) string))

func FetchGooglePlusPost(id, lastPost string) {
	return
}

// JPY returns exchange rate for each unit foreign currencies.
func JPY(args []string) string {
	return "hoge+1"
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
		}

	case "GET":
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err.Error())
		}
		log.Printf("Get request: %v", data)
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
		}
	}
	return
}

func handleEvents(events []Event) (results string) {
	for _, ev := range events {
		results += execCommand(ev)
	}
	return
}
