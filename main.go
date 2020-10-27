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

	commands := []tgbotapi.BotCommand{
		{
			Command:     "status",
			Description: "Returns the server status",
		},
	}

	if err := bot.SetMyCommands(commands); err != nil {
		log.Panic(err)
	}

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

		cmd := update.Message.Command()

		if cmd == "status" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, status())
			msg.ReplyToMessageID = update.Message.MessageID
			if _, err := bot.Send(msg); err != nil {
				log.Printf("Error sending msg %s : %v", msg.Text, err)
			}
			continue
		}
	}
}

func refresh() ([]byte, error) {
	resp, err := http.Get(api + "/servers")

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close() // nolint: errcheck

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func status() string {
	var str strings.Builder

	for _, serv := range servers {
		resp, err := http.Get(api + "/server/" + serv)
		if err != nil {
			str.WriteString(serv)
			str.WriteString(": Error (")
			str.WriteString(resp.Status)
			str.WriteString(")\n")
			continue
		}

		defer resp.Body.Close() // nolint: errcheck

		if resp.StatusCode == http.StatusServiceUnavailable {
			str.WriteString(serv)
			str.WriteString(": Offline\n")
			continue
		}

		b, _ := ioutil.ReadAll(resp.Body)

		s := struct {
			Online int64
			Max    int64
		}{}
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
