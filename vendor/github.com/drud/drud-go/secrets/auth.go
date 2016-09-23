package secrets

import (
	"fmt"

	"github.com/howeyc/gopass"
)

// GetCredentials prompts user to username, password and returns them
func GetCredentials() (string, string) {
	var username string
	fmt.Println("Username:")
	//read value from terminal into username var
	fmt.Scanf("%s", &username)
	fmt.Printf("Password:\n")
	//get user's password without echoing to terminal
	pass := GetMaskedInput()

	return username, string(pass)
}

// GetOTP prompts user for 2fa token
func GetOTP() (otp string) {
	fmt.Println("2fa token:")
	fmt.Scanf("%s", &otp)

	return
}

// GetMaskedInput gets a string from prompt to user but masks the input from the terminal
func GetMaskedInput() string {
	pass, _ := gopass.GetPasswdMasked()
	return string(pass)
}
