package utils

type OraclePriceResponse struct {
	Data map[string]struct {
		Price struct {
			Type string `json:"type"`
			Hex  string `json:"hex"`
		} `json:"price"`
		Expo        int `json:"expo"`
		PublishTime int `json:"publishTime"`
	} `json:"data"`
	Tokens  []string `json:"tokens"`
	Message string   `json:"message"`
}
