package test

import (
	"app/utils"
	"fmt"
	"math/rand"
	"os"
	"strings"
)

// pass
func TestCreateZimaOSDir() error {
	x := fmt.Sprint(rand.Int63())
	path := fmt.Sprintf("/DATA/%s", x)
	return utils.CreateZimaOSDir(path)
}

// pass
func TestMapZimaOSDir(path string, acl *utils.FolderACL) error {
	return utils.MapZimaOSDir(path, acl)
}

// pass
func TestUploadFile(file string, path string) error {
	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	return utils.UploadFile(f, 0, path)
}

// pass
func TestRemoteFileUploadFile(path1 string, path2 string) error {
	f, s, err := utils.GetRemoteFile(path1)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	return utils.UploadFile(f, s, path2)
}

// pass
func TestListRemoteDir(path string) error {
	Allocate()
	TestLoginForAuth()
	l1,l2, err := utils.ListRemoteDir(path)
	if err != nil {
		return err
	}
	fmt.Println(strings.Join(l1, "\n"))
	fmt.Println(strings.Join(l2, "\n"))
	return nil
}

func TestGetSynologyACL(path string) error {
	Allocate()
	TestLoginForAuth()
	utils.GetSynologyUser()
	utils.GetSynologyGroup()
	acl, err := utils.GetSynologyACL(path)
	if err != nil {
		return err 
	}
	for _,u := range acl.Auth {
		fmt.Printf("%s | Read: %d | Write: %d\n", u.User, u.Read, u.Write)
	}
	return nil
}
