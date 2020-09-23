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

type server struct {
	Online uint `json:"online"`
	Max    uint `json:"max"`
}

var api string
var servers []string

func init() {
	api = os.Getenv("API_URL")
	if len(api) == 0 {
		log.Panic("Empty api url")
	}

	b, err := refresh()
	if err != nil {
		log.Panic(err)
	}

	if err := json.Unmarshal(b, &servers); err != nil {
		log.Panic(err)
	}
}

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

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

		if !update.Message.IsCommand() {
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		switch update.Message.Command() {
		case "status":
			msg.Text = status()
		case "refresh":
			if b, err := refresh(); err == nil {
				if err := json.Unmarshal(b, &servers); err == nil {
					msg.Text = "Server: " + strings.Join(servers, ",")
				} else {
					msg.Text = "Unmarshal error"
				}
			} else {
				msg.Text = "Unable to refresh"
			}
		default:
			msg.Text = "I don't know that command"
		}

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending msg %s : %v", msg.Text, err)
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

func status() string {
	var str strings.Builder

	for _, serv := range servers {
		resp, err := http.Get(api + "/serben/" + serv)
		if err != nil {
			str.WriteString(serv)
			str.WriteString(": Error (")
			str.WriteString(resp.Status)
			str.WriteString(")\n")
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusServiceUnavailable {
			str.WriteString(serv)
			str.WriteString(": Offline\n")
			continue
		}

		b, _ := ioutil.ReadAll(resp.Body)

		var s server
		if err := json.Unmarshal(b, &s); err != nil {
			str.WriteString(serv)
			str.WriteString(": Unmarshal error")
			continue
		}

		str.WriteString(serv)
		str.WriteString(": ")
		str.WriteString(strconv.FormatUint(uint64(s.Online), 10))
		str.WriteString("/")
		str.WriteString(strconv.FormatUint(uint64(s.Max), 10))
		str.WriteString("\n")
	}

	return str.String()
}
