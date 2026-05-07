package commands

import (
	"encoding/json"
	"fmt"
	"net/http"

	shared "mangahub/cmd/cli/commands/shared"
)

func getUserInfoFromToken(token string) (struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}, error) {
	var userInfo struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}

	req, _ := http.NewRequest(http.MethodGet, shared.APIURL("/users/me"), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return userInfo, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return userInfo, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return userInfo, err
	}
	return userInfo, nil
}
