package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type SynologyUser struct {
	Username string 
	UID int 
	GID int 
	Admin bool
	// HomeDir string 
	// Shell string 
}



func SshConnect() error {

	// set global settings
	// SshHost = host
	// SshPort = port
	// SshUser = user
	// SshPass = pass

	config = &ssh.ClientConfig{
		User: SshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(SshPass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	// connection test
	addr := fmt.Sprintf("%s:%d", SshHost, SshPort)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return err
	}
	SshClient = client
	return nil
}

func GetSynologyUser() error {
	cmd := "cat /etc/passwd"
	cmd1 := "cat /etc/group"
	var admins []string
	UsersMap = make(map[string]SynologyUser)
	session1, err := SshClient.NewSession()
	if err != nil {
		
		return err
	}
	defer session1.Close()

	output1, err := session1.CombinedOutput(cmd1)
	if err != nil {
		
		return err
	}
	lines1 := strings.Split(strings.TrimSpace(string(output1)), "\n")
	for _, line1 := range lines1 {
		if len(line1) == 0 {
			continue
		}
		fields1 := strings.Split(line1, ":")
		groupname := fields1[0]
		if groupname == "administrators" {
			admins = strings.Split(strings.TrimSpace(fields1[3]), ",")
			break
		} else {
			continue
		}
	}

	session, err := SshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		fields := strings.Split(line, ":")

		username := fields[0]
		uidStr := fields[2]
		gidStr := fields[3]
		uid, _ := strconv.Atoi(uidStr)
		gid, _ := strconv.Atoi(gidStr)
		// not include synology admin account
		if uid > 1024 && gid == 100 && username != "guest" {
			isadmin := false
			for _, user := range admins {
				if user == username {
					isadmin = true
					break
				}
			}
			user := SynologyUser{
				Username: username,
				UID:      uid,
				GID:      gid,
				Admin:    isadmin,
			}
			// Users = append(Users, user)
			UsersMap[username] = user
		} else {
			continue
		}
	}
	return nil
}


func GetSynologyGroup() error {
	cmd := "cat /etc/group"

	GroupsMap = make(map[string][]string)
	session, err := SshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close() 
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue 
		}
		fields := strings.Split(line, ":")
		if len(fields) < 4 {
			continue
		}
		groupname := fields[0]
		gidStr := fields[2]
		gid, _ := strconv.Atoi(gidStr)
		users := strings.Split(fields[3],",")
		if gid > 65535 {
			var humanUsers []string 
			for _, u := range users {
				if _, ok := UsersMap[u]; ok {
					humanUsers = append(humanUsers,u)
				}
			}
			
			if len(humanUsers) > 0 {
				GroupsMap[groupname] = humanUsers
			}
		}
	}
	return nil
}


func CreateZimaUser() error {
	for u, _ := range UsersMap {
		 err := MapZimaUser(u)
		if err != nil {
			return err
		}
	}
	return nil
}

func MapZimaUser(username string) error {
	url := fmt.Sprintf("http://%s/v2/users/samba/user", ZimaHost)

	type ZimaUser struct {
		User string `json:"username"`
		Pass string `json:"password"`	
	}

	reqBody := ZimaUser {
		User : username,
		Pass : "123456",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("json marshal failed: %v", ZimaHost)
		
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("create request failed: %v", err)
		}

		req.Header.Set("Authorization", Authorization)
		req.Header.Set("Content-Type","application/json")
		client := &http.Client{}
		resp, err := client.Do(req)

		if err != nil {
		fmt.Printf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body) // 读取错误信息以便调试
		fmt.Printf("api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}
	bodyBytes, _ := io.ReadAll(resp.Body)

	//  打印调试
	fmt.Println("调试响应内容:", string(bodyBytes))
	// fmt.Println(resp.StatusCode)
	return nil
}