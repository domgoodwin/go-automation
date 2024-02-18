package homeassistant

import (
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

func CreateClient() *Client {
	token := os.Getenv("HA_TOKEN")
	return &Client{
		accessToken: token,
		url:         "homeassistant.local:8123",
		httpClient:  &http.Client{},
	}
}

type Client struct {
	accessToken string
	url         string
	wsc         *websocket.Conn
	wsType      int
	httpClient  *http.Client
}
