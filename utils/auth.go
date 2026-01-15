package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type loginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func LoginForAuth() error {

	url := fmt.Sprintf("http://%s/v1/users/login", ZimaHost)
	payload := loginPayload{
		Username: ZimaUsername,
		Password: ZimaPassword,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil
	}

	defer resp.Body.Close()
	// if not 200, ...
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, _ := io.ReadAll(resp.Body)

	var loginResp struct {
		Success int `json:"success"` // 可选：加上这个方便判断业务是否成功
		Data    struct {
			Token struct {
				RefreshToken string `json:"refresh_token"`
			} `json:"token"`
		} `json:"data"`
	}

	if err := json.Unmarshal(bodyBytes, &loginResp); err != nil {
		return err
	}

	Authorization = loginResp.Data.Token.RefreshToken

	return nil

}
