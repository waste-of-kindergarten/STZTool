package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// 对应 JSON 中的 "extensions" 部分
type ZimaExtension struct {
	Health   bool  `json:"health"`
	Shortage bool  `json:"shortage,omitempty"` // omitempty: 如果JSON里没有这个字段，则忽略
	Size     int64 `json:"size"`               // 使用 int64 防止溢出
	Used     int64 `json:"used"`
}

// 对应 JSON 中的每一个磁盘对象
type ZimaDisk struct {
	Extensions ZimaExtension `json:"extensions"`
	Font       string        `json:"font"`
	Name       string        `json:"name"`
	Path       string        `json:"path"`
	Type       string        `json:"type"`
}

func GetZimaDrive() ([]ZimaDisk, error) {
	url := fmt.Sprintf("http://%s/v2/local_storage/storages", ZimaHost)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Authorization", Authorization)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求发送失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API Error (Status Code %d): %s", resp.StatusCode, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %v", err)
	}

	var disks []ZimaDisk
	if err := json.Unmarshal(body, &disks); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %v. 原始内容: %s", err, string(body))
	}
	return disks, nil
}

type Volume struct {
	Size  int64  `json:"size"`  
	Used  int64  `json:"used"`  
	Avail int64  `json:"avail"`  
	Usedr string `json:"usedr"`  
	Mount string `json:"id"` 
}

func GetSynologyVolume() ([]Volume, error) {
	cmd := "df | grep '/volume' | awk '{print $2, $3, $4, $5, $6}'"
	var vol []Volume
	session, err := SshClient.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		fields := strings.Split(line, " ")

		sizeKB, _ := strconv.ParseInt(fields[0], 10, 64)
		usedKB, _ := strconv.ParseInt(fields[1], 10, 64)
		availKB, _ := strconv.ParseInt(fields[2], 10, 64)
		vol = append(vol, Volume{
			Size:  sizeKB * 1024,
			Used:  usedKB * 1024,
			Avail: availKB * 1024,
			Usedr: fields[3],
			Mount: fields[4],
		})
	}

	return vol, nil
}
