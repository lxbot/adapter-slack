package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"github.com/labstack/echo/v4"
	"github.com/lxbot/lxlib"
	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
	"log"
	"net/http"
	"os"
	"strings"
)

type M = map[string]interface{}

var ch *chan M
var client *slack.Client
var me *slack.AuthTestResponse
var token string
var secret string

func Boot(c *chan M) {
	ch = c
	gob.Register(slackevents.MessageEvent{})
	gob.Register(slackevents.AppMentionEvent{})
	gob.Register(slackevents.File{})
	gob.Register(slackevents.Edited{})
	gob.Register(slackevents.Icon{})

	token = os.Getenv("LXBOT_SLACK_OAUTH_ACCESS_TOKEN")
	if token == "" {
		log.Fatalln("invalid token:", "'LXBOT_SLACK_OAUTH_ACCESS_TOKEN' にアクセストークンを設定してください")
	}
	secret = os.Getenv("LXBOT_SLACK_SIGNING_SECRET")
	if secret == "" {
		log.Fatalln("invalid signing secret:", "'LXBOT_SLACK_SIGNING_SECRET' に署名シークレットを設定してください")
	}
	client = slack.New(
		token,
		slack.OptionDebug(true),
		slack.OptionLog(log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)),
	)
	i, err := client.AuthTest()
	if err != nil {
		log.Fatalln(err)
	}
	me = i
	log.Println("bot user:", *me)

	go listen()

	log.Println(client.SetUserPresence("active"))
}

func Send(msg M) {
	m, err := lxlib.NewLXMessage(msg)
	if err != nil {
		log.Println(err)
		return
	}

	texts := split(m.Message.Text, 50000)
	for _, v := range texts {
		_, _, err := client.PostMessageContext(context.Background(), m.Room.ID, slack.MsgOptionText(v, false))
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func Reply(msg M) {
	m, err := lxlib.NewLXMessage(msg)
	if err != nil {
		log.Println(err)
		return
	}

	texts := split(m.Message.Text, 50000)
	for _, v := range texts {
		_, _, err := client.PostMessageContext(context.Background(), m.Room.ID, slack.MsgOptionText("<@"+m.User.ID+"> "+v, false))
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func listen() {
	e := echo.New()
	e.GET("/", get)
	e.POST("/", post)
	_ = e.Start("0.0.0.0:1323")
}

func get(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}

func post(c echo.Context) error {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(c.Request().Body); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	body := buf.Bytes()

	sv, err := slack.NewSecretsVerifier(c.Request().Header, secret)
	if err != nil {
		log.Println(err)
		return c.NoContent(http.StatusUnauthorized)
	}
	if _, err := sv.Write(body); err != nil {
		log.Println(err)
		return c.NoContent(http.StatusUnauthorized)
	}
	if err := sv.Ensure(); err != nil {
		log.Println(err)
		return c.NoContent(http.StatusUnauthorized)
	}

	// 動かん　上のSecretsVerifierでやる
	// eventsAPIEvent, err := slackevents.ParseEvent(body, slackevents.OptionVerifyToken(&slackevents.TokenComparator{}))

	eventsAPIEvent, err := slackevents.ParseEvent(body, slackevents.OptionNoVerifyToken())
	if err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		if err := json.Unmarshal(body, &r); err != nil {
			log.Println(err)
			return c.NoContent(http.StatusInternalServerError)
		}
		return c.JSON(http.StatusOK, map[string]string{
			"challenge": r.Challenge,
		})
	}
	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent
		switch event := innerEvent.Data.(type) {
		case *slackevents.MessageEvent:
			go onMessage(event)
		case *slackevents.AppMentionEvent:
			go onAppMention(event)
		}
		return c.NoContent(http.StatusOK)
	}
	return c.NoContent(http.StatusNotAcceptable)
}

func onMessage(event *slackevents.MessageEvent) {
	if event.User == me.UserID {
		return
	}

	text := strings.TrimSpace(event.Text)

	attachments := make([]M, len(event.Files))
	for i, v := range event.Files {
		attachments[i] = M{
			"url": v.URLPrivate,
			"description": v.Title,
		}
	}

	chnnnelName := event.Channel
	topic := ""
	if i, err := client.GetChannelInfo(event.Channel); err == nil {
		chnnnelName = i.Name
		topic = i.Topic.Value
	}

	*ch <- M{
		"user": M{
			"id":   event.User,
			"name": event.Username,
		},
		"room": M{
			"id":          event.Channel,
			"name":        chnnnelName,
			"description": topic,
		},
		"message": M{
			"id":          event.TimeStamp,
			"text":        text,
			"attachments": attachments,
		},
		"is_reply":  false,
		"channel_type": event.ChannelType,
		"thread_ts": event.ThreadTimeStamp, // TODO
		"raw":       event,
	}
}

func onAppMention(event *slackevents.AppMentionEvent) {
	if event.User == me.UserID {
		return
	}

	text := strings.TrimSpace(event.Text)

	chnnnelName := event.Channel
	topic := ""
	if i, err := client.GetChannelInfo(event.Channel); err == nil {
		chnnnelName = i.Name
		topic = i.Topic.Value
	}
	userName := ""
	if u, err := client.GetUserInfo(event.User); err == nil {
		userName = u.Name
	}

	*ch <- M{
		"user": M{
			"id":   event.User,
			"name": userName,
		},
		"room": M{
			"id":          event.Channel,
			"name":        chnnnelName,
			"description": topic,
		},
		"message": M{
			"id":          event.TimeStamp,
			"text":        text,
			"attachments": make([]M, 0),
		},
		"is_reply":  false,
		"channel_type": "app_mention",
		"thread_ts": event.ThreadTimeStamp, // TODO
		"raw":       event,
	}
}

func split(s string, n int) []string {
	result := make([]string, 0)
	runes := bytes.Runes([]byte(s))
	tmp := ""
	for i, r := range runes {
		tmp = tmp + string(r)
		if (i+1)%n == 0 {
			result = append(result, tmp)
			tmp = ""
		}
	}
	return append(result, tmp)
}
