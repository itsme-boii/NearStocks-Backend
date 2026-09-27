package dtos

import (
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
)

type ZealyQuestClaimDto struct {
	UserId      string       `json:"userId"`
	CommunityId string       `json:"communityId"`
	Subdomain   string       `json:"subdomain"`
	QuestId     string       `json:"questId"`
	RequestId   string       `json:"requestId"`
	Accounts    *AccountsDto `json:"accounts,omitempty"`
}

type AccountsDto struct {
	Email   string      `json:"email,omitempty"`
	Wallet  string      `json:"wallet,omitempty"`
	Discord *DiscordDto `json:"discord,omitempty"`
	Twitter *TwitterDto `json:"twitter,omitempty"`
}

type DiscordDto struct {
	Id     string `json:"id"`
	Handle string `json:"handle"`
}

type TwitterDto struct {
	Id       string `json:"id"`
	Username string `json:"username"`
}

type InteractDto struct {
	Address  string `json:"address" binding:"required"`
	Twitter  string `json:"twitter,omitempty"`
	Discord  string `json:"discord,omitempty"`
	Telegram string `json:"telegram,omitempty"`
	Email    string `json:"email,omitempty"`
}
type LeaderboardAllResult struct {
	EthAddress          string
	TotalTrades         int
	WinningTrades       int
	LosingTrades        int
	ReduceZeroPnlTrades int
	TotalRealizedPnl    *ctypes.BigInt
	UserName            *string
}

type LeaderboardResult struct {
	UserAddress         string
	TotalTrades         int
	WinningTrades       int
	LosingTrades        int
	ReduceZeroPnlTrades int
	TotalRealizedPnl    *ctypes.BigInt
}
