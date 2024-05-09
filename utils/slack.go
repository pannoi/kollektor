package utils

import (
	"github.com/slack-go/slack"
)

func SendSlackMessage(webhookUrl string, title string, text string) error {
	attachment := slack.Attachment{
		Color: "#00ADD8",
		Title: title,
		Text:  text,
	}
	msg := slack.WebhookMessage{
		Attachments: []slack.Attachment{attachment},
	}

	err := slack.PostWebhook(webhookUrl, &msg)
	if err != nil {
		return err
	}
	return nil
}
