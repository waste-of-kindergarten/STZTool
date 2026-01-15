package test

import (
	"app/utils"
	"fmt"
)

// pass
func TestLoginForAuth() error{
	err := utils.LoginForAuth()
	if err != nil {
		return err
	} else {
		fmt.Println(utils.Authorization)
		return nil
	}
}

