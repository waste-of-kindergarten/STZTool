package main

import (
	"app/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"path/filepath"

	// "fmt"

	"golang.org/x/crypto/ssh"
)

var sshHost string
var sshPort int
var sshUser string
var sshPass string

var config *ssh.ClientConfig
var sshClient *ssh.Client

var Users []utils.SynologyUser

func walk(remoteRoot string, localRoot string, create bool) error {
	fmt.Printf("walking on %s\n", remoteRoot)
	acls, err := utils.GetSynologyACL(remoteRoot)
	if err != nil {
		return err
	}

	if !create {
			err = utils.CreateZimaOSDir(localRoot)
			if err != nil {
				return err
			}
	}

	err = utils.MapZimaOSDir(localRoot, acls)
	if err != nil {
		return err
	}
	folders, files, err := utils.ListRemoteDir(remoteRoot)
	for _, file := range files {
		fullRemoteDir := filepath.Join(remoteRoot, file)
		fullLocalDir := filepath.Join(localRoot, file)
		f, size, err := utils.GetRemoteFile(fullRemoteDir)
		if err != nil {
			return nil
		}
		err = utils.UploadFile(f, size, fullLocalDir)
		if err != nil {
			return nil
		}
	}
	for _, folder := range folders {
		fullRemoteDir := filepath.Join(remoteRoot, folder)
		fullLocalDir := filepath.Join(localRoot, folder)
		err = walk(fullRemoteDir, fullLocalDir, false)
		if err != nil {
			return err
		}
	}
	return nil
}

func allocate(zimaHost string, zimaUsername string, zimaPassword string,
	sshHost string, sshPort int, sshUser string, sshPass string) {
	utils.ZimaHost = zimaHost
	utils.ZimaUsername = zimaUsername
	utils.ZimaPassword = zimaPassword

	utils.SshHost = sshHost
	utils.SshPort = sshPort
	utils.SshUser = sshUser
	utils.SshPass = sshPass

	utils.SshConnect()
}

type ConfigRequest struct {
	Synology struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"synology"`
	Zima struct {
		Host     string `json:"host"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"zima"`
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	// ==========================================
	// 1. 核心：处理 CORS 跨域 (必须！)
	// ==========================================
	// 允许任何来源访问 (生产环境建议改为具体的 http://localhost:5173)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// 允许前端发送的 Headers
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	// 允许的方法
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	// 处理浏览器的预检请求 (Preflight OPTIONS request)
	// 浏览器在发送 POST 之前会先发一个 OPTIONS 询问服务器是否允许跨域
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// ==========================================
	// 2. 处理实际的 POST 请求
	// ==========================================
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 读取 Body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// (可选) 解析并打印出来看看
	var reqData ConfigRequest
	if err := json.Unmarshal(body, &reqData); err != nil {
		fmt.Println("解析 JSON 失败:", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	utils.SshHost = reqData.Synology.Host
	utils.SshUser = reqData.Synology.Username
	utils.SshPass = reqData.Synology.Password

	utils.SshPort = reqData.Synology.Port

	utils.ZimaHost = reqData.Zima.Host
	utils.ZimaUsername = reqData.Zima.Username
	utils.ZimaPassword = reqData.Zima.Password

	err = utils.SshConnect()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "fail", "message": "SSH connection failed"}`))
		panic(err)
	}

	err = utils.LoginForAuth()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "fail", "message": "Zima login failed"}`))
		panic(err)
	}

	err = utils.GetSynologyUser()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "fail", "message": "Synology User getting failed"}`))
		panic(err)
	}

	err = utils.GetSynologyGroup()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "fail", "message": "Synology group getting failed"}`))
		panic(err)
	}

	// 在控制台打印接收到的数据 (方便你调试)
	fmt.Printf("收到前端配置:\n")
	fmt.Printf("  [群晖] %s@%s:%d\n", reqData.Synology.Username, reqData.Synology.Host, reqData.Synology.Port)
	fmt.Printf("  [Zima] %s@%s\n", reqData.Zima.Username, reqData.Zima.Host)

	// ==========================================
	// 3. 直接回复 200 OK
	// ==========================================
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "message": "Configuration received"}`))
}

func synologyStorageHandler(w http.ResponseWriter, r *http.Request) {
	// === 关键修改：必须先设置 CORS 头 ===
	setupCORS(w)

	// 处理浏览器的预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	fmt.Println("📂 [Synology] 正在通过 SSH 获取数据...")

	// 使用全局配置中的账号密码
	vols, err := utils.GetSynologyVolume()
	prettyJSON, err := json.MarshalIndent(vols, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println("synology")
	fmt.Println(string(prettyJSON))
	if err != nil {
		fmt.Printf("❌ 群晖获取失败: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vols)
}

func zimaStorageHandler(w http.ResponseWriter, r *http.Request) {
	// === 关键修改：必须先设置 CORS 头 ===
	setupCORS(w)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	fmt.Println("📂 [ZimaOS] 正在通过 API 获取数据...")

	// 使用全局配置中的信息
	disks, err := utils.GetZimaDrive()
	prettyJSON, _ := json.MarshalIndent(disks, "", "  ")
	fmt.Println(string(prettyJSON))
	if err != nil {
		fmt.Printf("❌ ZimaOS 获取失败: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(disks)
}

type MigrationRequest struct {
	// key是源路径(群晖), value是目标路径(Zima)
	Mappings map[string]string `json:"mappings"`
}

// ... 之前的 Handler ...

// 2. 新增：处理开始迁移的请求 [POST] /migrate
func startMigrationHandler(w http.ResponseWriter, r *http.Request) {
	utils.Done = false
	utils.ErrorLog = nil
	utils.UploadTime = 0
	utils.UploadFilesTotal = 0
	setupCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 解析请求体
	body, _ := io.ReadAll(r.Body)
	var req MigrationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "无法解析请求数据", http.StatusBadRequest)
		return
	}

	// 打印出来验证一下
	fmt.Println("🚀 [Task] 收到迁移任务指令！")
	for src, dst := range req.Mappings {
		fmt.Printf("   - 计划将 %s 迁移至 %s\n", src, dst)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "started", "message": "Task initiated"}`))

	// TODO: 这里以后会启动真正的后台 goroutine 任务
	// go StartCopyTask(req.Mappings)
	go func(tasks map[string]string) {
		startTime := time.Now()
		utils.CreateZimaUser()
		for v, d := range tasks {
			err := walk(v, d, true)
			if err != nil {
				utils.ErrorLog = append(utils.ErrorLog, err.Error())
			}
		}
		utils.UploadTime = int64(time.Since(startTime).Seconds())
		println(utils.UploadTime)
		println(utils.UploadFilesTotal)
		utils.Done = true
	}(req.Mappings)

}

func getProgressHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 必须设置 CORS，否则前端拿不到数据
	setupCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 2. 构造假数据
	// 注意：这里的字段名 (json:"...") 必须和前端使用的变量名完全一致
	var ratio int

	current := atomic.LoadInt64(&utils.UploadCurrent)
	total := atomic.LoadInt64(&utils.UploadTotal)

	if !utils.Done {
		if total != 0 {
			ratio = int(float64(current) / float64(total) * 100)
		} else {
			ratio = 100
		}

		data := struct {
			Status      string `json:"status"`
			Percentage  int    `json:"percentage"`
			CurrentFile string `json:"current_file"`
			Message     string `json:"message"`
		}{
			Status:      "running",
			Percentage:  ratio, // 你可以手动改这个数字来测试进度条变化
			CurrentFile: utils.UploadFileName,
			Message:     "正在传输中",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	} else {
		ratio = 100
		data := struct {
			Status      string   `json:"status"`
			Percentage  int      `json:"percentage"`
			CurrentFile string   `json:"current_file"`
			Message     string   `json:"message"`
			UploadTime  int64    `json:"uploadTime"`
			TotalSize   int64    `json:"totalSize"`
			Errors      []string `json:"errors,omitempty"`
		}{
			Status:      "finished",
			Percentage:  ratio, // 你可以手动改这个数字来测试进度条变化
			CurrentFile: "",
			Message:     "传输完成",
			UploadTime:  utils.UploadTime,       // 从 utils 获取计算好的时间
			TotalSize:   utils.UploadFilesTotal, // 从 utils 获取累计的大小
			Errors:      utils.ErrorLog,
		}
		println(utils.UploadTime)
		println(utils.UploadFilesTotal)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}

	// 3. 发送 JSON

}

func setupCORS(w http.ResponseWriter) {
	// 允许任何来源访问 (或者指定你的前端地址 http://localhost:5173)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// 允许前端发送的 Headers (Content-Type, Authorization 等)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	// 允许的方法
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
}

func main() {

	http.HandleFunc("/migrate", startMigrationHandler)
	http.HandleFunc("/config", configHandler)
	http.HandleFunc("/progress", getProgressHandler)
	http.HandleFunc("/storage/synology", synologyStorageHandler)
	http.HandleFunc("/storage/zima", zimaStorageHandler)
	fmt.Println("🚀 后端服务已启动，监听端口 :5175")
	fmt.Println("👉 等待前端发送 POST 请求到 http://localhost:5175/config ...")
	fmt.Println("   POST /config")
	fmt.Println("   GET  /storage/synology")
	fmt.Println("   GET  /storage/zima")
	// 启动服务
	if err := http.ListenAndServe(":5175", nil); err != nil {
		fmt.Printf("启动失败: %s\n", err)
	}

}
