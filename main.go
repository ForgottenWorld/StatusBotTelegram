package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Server struct {
	Online uint `json:"online"`
	Max    uint `json:"max"`
}

var api string

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	api = os.Getenv("API_URL")
	if len(api) == 0 {
		log.Panic("Empty api url")
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	b, err := refresh()
	if err != nil {
		log.Panic(err)
	}

	var servers []string
	json.Unmarshal(b, &servers)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			switch update.Message.Command() {
			case "status":

				var str strings.Builder
				for _, serv := range servers {

					resp, err := http.Get(api + "/serben/" + serv)
					if err != nil {
						str.WriteString(serv)
						str.WriteString(": Error (")
						str.WriteString(resp.Status)
						str.WriteString(")\n")
					} else if resp.StatusCode == http.StatusServiceUnavailable {
						resp.Body.Close()
						str.WriteString(serv)
						str.WriteString(": Offline\n")
					} else {
						b, err = ioutil.ReadAll(resp.Body)
						resp.Body.Close()
						var s Server
						json.Unmarshal(b, &s)
						str.WriteString(serv)
						str.WriteString(": ")
						str.WriteString(strconv.FormatUint(uint64(s.Online), 10))
						str.WriteString("/")
						str.WriteString(strconv.FormatUint(uint64(s.Max), 10))
						str.WriteString("\n")
					}
				}
				msg.Text = str.String()
			case "refresh":
				b, err := refresh()
				if err != nil {
					msg.Text = "Unable to refresh"
				} else {
					json.Unmarshal(b, &servers)
					msg.Text = "Server: " + strings.Join(servers, ",")
				}
			default:
				msg.Text = "I don't know that command"
			}
			bot.Send(msg)
		}
	}
}

func refresh() ([]byte, error) {
	resp, err := http.Get(api + "/servers")

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return b, nil
}
