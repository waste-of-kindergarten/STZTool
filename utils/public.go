package utils

import "golang.org/x/crypto/ssh"

var ZimaHost string
var ZimaUsername string 
var ZimaPassword string

var Authorization string

var SshHost string
var SshPort int
var SshUser string
var SshPass string

var config *ssh.ClientConfig
var SshClient *ssh.Client

var UsersMap map[string]SynologyUser
var GroupsMap map[string] []string

var UploadTotal   int64 
var UploadCurrent int64 
