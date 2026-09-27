package xclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"net/http"
	"os"
	"strings"
)

type DiscordClient struct {
	baseUrl string
}

// Verify that the DiscordClient implements the XClient interface
var _ XClient = &DiscordClient{}

var GlobalDiscordClient *DiscordClient

func InitDiscordClient() {
	xlog.Infof("Initializing Discord Client")
	GlobalDiscordClient = NewDiscordClient()
}


func GetGlobalDiscordClient() *DiscordClient {
	if GlobalDiscordClient == nil {
		InitDiscordClient()
	}
	return GlobalDiscordClient
}

func NewDiscordClient() *DiscordClient {
	baseUrl := os.Getenv("DISCORD_NETWORK_SYNC_WEBHOOK")
	if baseUrl == "" {
		baseUrl = "https://discord.com/api/webhooks/1266392465934516335/NUxUsrcaSO8zPJGN5444z33RfRyPZrnxVWrsHFxnJsSgGJWpXeP0V8t8b66Uds9tkCcC"
	}
	return &DiscordClient{baseUrl: baseUrl}
}

func (dc *DiscordClient) GetBaseUrl() string {
	return dc.baseUrl
}

type DiscordWebhookSync struct {
	Content string `json:"content"`
}

var DiscordMemberIds = strings.Join([]string{"<@725368429703463002>", "<@681087629617266829>"}, " ")
var DiscordMemberIdsWithDark = strings.Join([]string{"<@859825849251397682>", "<@725368429703463002>", "<@681087629617266829>"}, " ")

func (dc *DiscordClient) SendWebhookMessage(message string) error {
	return dc.SendWebhookMessageWithMentions(message, false)
}

func (dc *DiscordClient) SendWebhookMessageWithMentions(message string, includeDark bool) error {
	var memberIds string
	if includeDark {
		memberIds = DiscordMemberIdsWithDark
	} else {
		memberIds = DiscordMemberIds
	}
	
	requestBody := DiscordWebhookSync{Content: fmt.Sprintf("%v %+v", memberIds, message)}
	jsonPayload, err := json.Marshal(requestBody)
	if err != nil {
		xlog.Errorf("DW - error marshaling request body: %v", err)
		return err
	}
	resp, err := http.Post(dc.GetBaseUrl(), "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		xlog.Errorf("Error posting to discord: %v", err)
		return err
	}
	defer resp.Body.Close()
	return nil
}
