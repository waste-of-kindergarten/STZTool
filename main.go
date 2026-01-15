package main

import (
	"app/test"
	"app/utils"
	"fmt"

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

func walk(remoteRoot string, localRoot string) error {
	fmt.Printf("walking on %s\n", remoteRoot)
	acls, err := utils.GetSynologyACL(remoteRoot)
	if err != nil {
		return err
	}
	err = utils.CreateZimaOSDir(localRoot)
	if err != nil {
		return err
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
		err = walk(fullRemoteDir, fullLocalDir)
		if err != nil {
			return err
		}
	}
	return nil
}

// verified

// func walkAndMigrate(remoteRoot, localRoot string) error {
// 	return filepath.WalkDir(remoteRoot, func(path string, d fs.DirEntry, err error) error {
// 		if err != nil {
// 			return err
// 		}

// 		rel, _ := filepath.Rel(remoteRoot, path)
// 		target := filepath.Join(localRoot, rel)

// 		if d.IsDir() {
// 			acl, err := utils.GetFolderACL(path,Users)
// 			if err != nil {
// 				return err
// 			}
// 			return utils.MapZimaOSDir(target, acl)
// 		}

// 		info, _ := d.Info()
// 		fmt.Printf("→ FILE %s\n", path)

// 		if err := utils.ScpFileWithProgress(
// 			sshUser,
// 			sshHost,
// 			sshPort,
// 			path,
// 			target,
// 		); err != nil {
// 			return err
// 		}

// 		return utils.VerifyFile(info.Size(), target)
// 	})
// }

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

func main() {
	test.Allocate()
	err := utils.LoginForAuth()
	if err != nil {
		panic(err)
	}
	err = utils.GetSynologyUser()
	if err != nil {

		panic(err)
	}

	err = utils.GetSynologyGroup()
	if err != nil {

		panic(err)
	}
	utils.CreateZimaUser()
	err = walk("/volume1", "/media/ZimaOS-HD/volume1")
	if err != nil {
		panic(err)
	}

}
