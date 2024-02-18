package mastodon

import (
	"context"
	"os"

	"github.com/mattn/go-mastodon"
)

type Client struct {
	clientID     string
	clientSecret string
	url          string
	botEmail     string
	botPassword  string

	mClient *mastodon.Client
}

func SetupClient(ctx context.Context) (*Client, error) {
	client := &Client{
		url:          "https://botsin.space",
		clientID:     os.Getenv("CLIENT_ID"),
		clientSecret: os.Getenv("CLIENT_SECRET"),
		botEmail:     os.Getenv("BOT_EMAIL"),
		botPassword:  os.Getenv("BOT_PASSWORD"),
	}
	err := client.init(ctx)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) init(ctx context.Context) error {
	mClient := mastodon.NewClient(&mastodon.Config{
		Server:       c.url,
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
	})
	c.mClient = mClient
	err := c.mClient.Authenticate(ctx, c.botEmail, c.botPassword)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) UploadImage(ctx context.Context, file string) (string, error) {
	rsp, err := c.mClient.UploadMedia(ctx, file)
	if err != nil {
		return "", err
	}
	return string(rsp.ID), nil
}

func (c *Client) PostStatus(ctx context.Context, status string, mediaIDs []string) (string, error) {
	var mediaMastodonID []mastodon.ID
	for _, id := range mediaIDs {
		mediaMastodonID = append(mediaMastodonID, mastodon.ID(id))
	}
	post, err := c.mClient.PostStatus(ctx, &mastodon.Toot{
		Status:   status,
		MediaIDs: mediaMastodonID,
	})
	if err != nil {
		return "", err
	}
	return string(post.ID), nil
}
