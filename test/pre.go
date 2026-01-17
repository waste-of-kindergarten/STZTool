package test

import (
	"app/utils"
)

func Allocate() {
	utils.ZimaHost = "192.168.1.118"
	utils.ZimaUsername = "shihao"
	utils.ZimaPassword = "Cth-718281"

	utils.SshHost = "192.168.1.244"
	utils.SshPort = 22
	utils.SshUser = "shihao"
	utils.SshPass = "Cth-718281"

	utils.SshConnect()
}
