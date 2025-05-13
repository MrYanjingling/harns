package config

import (
	"lightiot/pkg/client"
	"lightiot/pkg/notification"
)

type Config struct {
	WeComClient  client.Client
	WeChatClient client.Client

	NotifyConfig notification.Config
}
