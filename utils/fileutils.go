package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"

	"path/filepath"

	"strings"

	"golang.org/x/crypto/ssh"
)

type FolderACL struct {
	Auth []UserACL
}

type UserACL_ struct {
	User  string
	Read  int // -1 for deny, 0 for no permission, 1 for allow
	Write int
}

type UserACL struct {
	User  string
	Read  bool
	Write bool
}

type FileTask struct {
	RemotePath string
	LocalPath  string
	Size       int64
}

type createFolderReq struct {
	Path string `json:"path"`
}

func MapZimaOSDir(path string, acl *FolderACL) error {
	url := fmt.Sprintf("http://%s/v2_1/files/share", ZimaHost)
	type ZimaFolder struct {
		Path       string    `json:"path"`
		Name       string    `json:"name"`
		TM         bool      `json:"tm"`
		Permission []UserACL `json:"permission"`
	}

	type CreateFolderReq []ZimaFolder

	reqBody := [1]ZimaFolder{{
		Path:       path,
		Name:       filepath.Base(path),
		TM:         false,
		Permission: acl.Auth,
	},
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
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body) // 读取错误信息以便调试
		fmt.Printf("api error (status %d): %s", resp.StatusCode, string(bodyBytes))
		fmt.Printf(">>> DEBUG Request Body: %s\n", string(jsonData))

	}
	bodyBytes, _ := io.ReadAll(resp.Body)

	//  打印调试
	fmt.Println("调试响应内容:", string(bodyBytes))
	// fmt.Println(resp.StatusCode)
	return nil
}

func CreateZimaOSDir(path string) error {

	url := fmt.Sprintf("http://%s/v2_1/files/folder", ZimaHost)
	reqBody := createFolderReq{
		Path: path,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("json marshal failed: %v", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request failed: %v", err)
	}

	// maybe add a test whether Authorization is expired
	req.Header.Set("Authorization", Authorization)
	req.Header.Set("Content-Type", "application/json")
	// 5. 发起请求
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// // 6. 检查响应状态码
	// // 通常 200 或 201 表示成功，这里假设非 200 系列都是错误
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

func VerifyFile(remoteSize int64, localPath string) error {

	info, err := os.Stat(localPath)
	if err != nil {
		return err
	}
	if info.Size() != remoteSize {
		return fmt.Errorf("size mismatch: %d vs %d", remoteSize, info.Size())
	}
	return nil
}

func UploadFile(reader io.Reader, size int64, path string) error {

	reader = NewProgressReader(reader, size)

	url := fmt.Sprintf("http://%s/v2_1/files/file/uploadV2", ZimaHost)
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer writer.Close()

		writer.WriteField("path", filepath.Dir(path))

		part, err := writer.CreateFormFile("file", filepath.Base(path))

		if err != nil {
			pw.CloseWithError(err)
			return
		}

		_, err = io.Copy(part, reader)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("copy file content failed: %v", err))
			return
		}
	}()

	req, err := http.NewRequest("POST", url, pr)
	if err != nil {
		return fmt.Errorf("create request failed: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", Authorization)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body) // 读取错误信息以便调试
		fmt.Printf("api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	fmt.Println(string(bodyBytes))
	return nil
}

func GetSynologyACL(path string) (*FolderACL, error) {
	useracls := []UserACL{}
	mapindexbyname := make(map[string]*UserACL_)

	cmd := fmt.Sprintf("/usr/syno/bin/synoacltool -get \"%s\"", path)
	session, err := SshClient.NewSession()
	if err != nil {

		return nil, err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		res := strings.TrimSpace(string(output))
		if !strings.HasPrefix(res, "(synoacltool.c, 596)It's Linux mode") {
			return nil, err
		} else { // Linux mode

			session1, err := SshClient.NewSession()
			if err != nil {
				return nil, err
			}
			defer session1.Close()
			cmd = fmt.Sprintf("stat -c \"%%A|%%U|%%G\" \"%s\"", path)
			output1, err := session1.CombinedOutput(cmd)
			if err != nil {

				return nil, err
			}

			line := string(output1)
			fields := strings.Split(line, "|")
			if len(fields) != 3 {

				return nil, fmt.Errorf("something unexpected happened")
			}

			permissions := fields[0]
			owner := fields[1]
			group := fields[2]

			if _, ok := UsersMap[owner]; ok {
				read := 0
				write := 0
				if permissions[1] == 'r' {
					read = 1
				}
				if permissions[2] == 'w' {
					write = 1
				}
				mapindexbyname[owner] = &UserACL_{
					User:  owner,
					Read:  read,
					Write: write,
				}
			}

			if vg, ok := GroupsMap[group]; ok {
				for _, n := range vg {
					if v, ok := mapindexbyname[n]; ok {
						if permissions[4] == 'r' {
							v.Read = 1
						}
						if permissions[5] == 'w' {
							v.Write = 1
						}
					} else {
						read := 0
						write := 0
						if permissions[4] == 'r' {
							read = 1
						}
						if permissions[5] == 'w' {
							write = 1
						}
						mapindexbyname[n] = &UserACL_{
							User:  n,
							Read:  read,
							Write: write,
						}
					}
				}
			}

			for _, acl := range mapindexbyname {
				acl1 := UserACL{
					User:  acl.User,
					Read:  acl.Read == 1,
					Write: acl.Write == 1,
				}
				useracls = append(useracls, acl1)
			}

			folderacls := &FolderACL{
				Auth: useracls,
			}

			return folderacls, nil

		}
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) != 6 {
			continue
		}
		ty := fields[0]
		name := fields[1]
		action := fields[2]
		permissions := fields[3]
		if strings.HasSuffix(ty, "user") {
			if v, ok := mapindexbyname[name]; ok {
				if action == "allow" {
					if permissions[0] == 'r' && v.Read == 0 {
						v.Read = 1
					}
					if permissions[1] == 'w' && v.Write == 0 {
						v.Write = 1
					}
				} else { // deny
					if permissions[0] == 'r' {
						v.Read = -1
					}
					if permissions[1] == 'w' {
						v.Write = -1
					}
				}
			} else { // initialize
				if _, ok := UsersMap[name]; ok { // index if in the humans usermap
					read := 0
					write := 0
					if action == "allow" {
						if permissions[0] == 'r' {
							read = 1
						}
						if permissions[1] == 'w' {
							write = 1
						}
					} else { // deny
						if action == "deny" {
							if permissions[0] == 'r' {
								read = -1
							}
							if permissions[1] == 'w' {
								write = -1
							}
						}
					}
					mapindexbyname[name] = &UserACL_{
						User:  name,
						Read:  read,
						Write: write,
					}
				}
			}
		} else if strings.HasSuffix(ty, "group") {
			if name == "administrators" {
				for n, u := range UsersMap {
					if u.Admin {
						if v, ok := mapindexbyname[name]; ok {
							if action == "allow" {
								if permissions[0] == 'r' && v.Read == 0 {
									v.Read = 1
								}
								if permissions[1] == 'w' && v.Write == 0 {
									v.Write = 1
								}
							} else { // deny
								if permissions[0] == 'r' {
									v.Read = -1
								}
								if permissions[1] == 'w' {
									v.Write = -1
								}
							}
						} else {
							read := 0
							write := 0
							if action == "allow" {
								if permissions[0] == 'r' {
									read = 1
								}
								if permissions[1] == 'w' {
									write = 1
								}
							} else { // deny
								if permissions[0] == 'r' {
									read = -1
								}
								if permissions[1] == 'w' {
									write = -1
								}
							}
							mapindexbyname[n] = &UserACL_{
								User:  n,
								Read:  read,
								Write: write,
							}
						}
					} else {
						continue
					}
				}
			} else if vg, ok := GroupsMap[name]; ok {
				for _, u := range vg { // u is username
					if v, ok := mapindexbyname[u]; ok {
						if action == "allow" {
							if permissions[0] == 'r' && v.Read == 0 {
								v.Read = 1
							}
							if permissions[1] == 'w' && v.Write == 0 {
								v.Write = 1
							}
						} else { // deny
							if permissions[0] == 'r' {
								v.Read = -1
							}
							if permissions[1] == 'w' {
								v.Write = -1
							}
						}
					} else {
						read := 0
						write := 0
						if action == "allow" {
							if permissions[0] == 'r' {
								read = 1
							}
							if permissions[1] == 'w' {
								write = 1
							}
						} else { // deny
							if permissions[0] == 'r' {
								read = -1
							}
							if permissions[1] == 'w' {
								write = -1
							}
						}
						mapindexbyname[u] = &UserACL_{
							User:  u,
							Read:  read,
							Write: write,
						}
					}
				}
			}
		} else {
			continue
		}

	}

	for _, acl := range mapindexbyname {
		acl1 := UserACL{
			User:  acl.User,
			Read:  acl.Read == 1,
			Write: acl.Write == 1,
		}

		if acl1.Read || acl1.Write {
			useracls = append(useracls, acl1)
		}
		
	}
	folderacls := &FolderACL{
		Auth: useracls,
	}

	prettyJSON, err := json.MarshalIndent(folderacls, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(prettyJSON))

	return folderacls, nil

}

// verified
func GetRemoteFile(remotePath string) (io.ReadCloser, int64, error) {

	cmd := fmt.Sprintf("stat -c %%s \"%s\"", remotePath)
	session1, err := SshClient.NewSession()
	
	if err != nil {
		session1.Close()
		return nil, -1, err
	}
	defer session1.Close()
	output, err := session1.CombinedOutput(cmd)
	if err != nil {
		session1.Close()
		return nil, -1, err
	}

	cleanOutput := strings.TrimSpace(string(output))
	size, err := strconv.ParseInt(string(cleanOutput), 10, 64)
	if err != nil {
		session1.Close()
		panic(err)
	}

	session, err := SshClient.NewSession()
	
	if err != nil {
		session.Close()
		return nil, -1, err
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, -1, err
	}
	err = session.Start(fmt.Sprintf("cat \"%s\"", remotePath))
	if err != nil {
		session.Close()
		return nil, -1, err
	}

	reader := &sshFileReader{
		Reader:  stdout,
		session: session,
	}

	UploadFileName = remotePath
	fmt.Printf("file size is %d", size)
	return reader, size, nil
}

// pass
func ListRemoteDir(remotePath string) ([]string, []string, error) {
	var folders []string
	var files []string
	cmd := fmt.Sprintf("cd \"%s\"; stat -c \"%%A|%%n\" *", remotePath)
	session, err := SshClient.NewSession()
	if err != nil {
		session.Close()
		return nil, nil, err
	}
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		session.Close()
		return nil, nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		field := strings.Split(line, "|")
		if len(field) != 2 {
			continue
		}

		permissions := field[0]
		name := field[1]

		if strings.HasPrefix(name, "#") || strings.HasPrefix(name, "@") || strings.HasPrefix(name, ".") {
			continue
		}
		if strings.HasPrefix(permissions, "d") {
			folders = append(folders, name)
		} else {
			files = append(files, name)
		}
	}
	return folders, files, nil
}

type sshFileReader struct {
	io.Reader
	session *ssh.Session
}

func (r *sshFileReader) Close() error {
	return r.session.Close()
}

type ProgressReader struct {
	r io.Reader
}

func NewProgressReader(r io.Reader, size int64) *ProgressReader {
	atomic.StoreInt64(&UploadTotal, size)
	atomic.StoreInt64(&UploadCurrent, int64(0))
	return &ProgressReader{r: r}
}

func (p *ProgressReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	if n > 0 {
		atomic.AddInt64(&UploadCurrent, int64(n))
	}
	return n, err
}
