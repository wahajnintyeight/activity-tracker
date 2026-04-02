package discord

import (
	"github.com/tropicalshadow/rich-go/client"
)

type DiscordClient struct {
	appID  string
	client *client.Client
}

func NewDiscordClient(appID string) *DiscordClient {
	return &DiscordClient{
		appID:  appID,
		client: client.NewClient(),
	}
}

func (d *DiscordClient) Logout() {
	if d.client != nil && d.client.IsLogged() {
		d.client.Logout()
	}
}

func Logout() {
	c := client.NewClient()
	c.Logout()
}
