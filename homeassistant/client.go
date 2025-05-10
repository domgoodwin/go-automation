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
		url:         "homeassistant.tailce93f.ts.net:8123",
		httpClient:  &http.Client{},
		counter:     1,
	}
}

type Client struct {
	accessToken string
	url         string
	wsc         *websocket.Conn
	wsType      int
	httpClient  *http.Client
	counter     int
}
