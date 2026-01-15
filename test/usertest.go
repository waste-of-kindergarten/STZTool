package test

import (
	"app/utils"
	"fmt"
)

func TestgetSynologyUser() error {
	err := utils.GetSynologyUser()
	for username, user := range utils.UsersMap {
    fmt.Printf("key=%s, value=%+v\n", username, user)
	}
	return err
}

func TestgetSynologyGroup() error {
	err := utils.GetSynologyGroup()
	for groupname, username := range utils.GroupsMap {
		fmt.Printf("key=%s, value=%+v\n", groupname, username)
	}
	return err
}