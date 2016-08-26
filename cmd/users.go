package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/drud/drud-go/drudapi"
	"github.com/spf13/cobra"
)

// usersCmd represents the users command
var usersListCmd = &cobra.Command{
	Use:     "users",
	Aliases: []string{"user"},
	Short:   "List users.",
	Long:    `List users.`,
	Run: func(cmd *cobra.Command, args []string) {

		ul := &drudapi.UserList{}

		err := drudclient.Get(ul)
		if err != nil {
			log.Fatalln(err)
		}

		ul.Describe()
	},
}

var usersCreateCmd = &cobra.Command{
	Use:     "user [username]",
	Aliases: []string{"users"},
	Short:   "Create new user.",
	Long:    `Create new user. Returns new user's temporary auth token.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Please supply a username")
			os.Exit(1)
		}

		u := &drudapi.User{
			Username: args[0],
			Hashpw:   RandomString(15),
		}

		err := drudclient.Post(u)
		if err != nil {
			log.Fatalln(err)
		}

		// running get to retrieve the auth token...this may be deprecated
		err = drudclient.Get(u)
		if err != nil {
			log.Fatalln(err)
		}

		// @todo this token should become a vault token instead of a custom one
		fmt.Println(u.Token)

	},
}

var userDeleteCmd = &cobra.Command{
	Use:   "user [username]",
	Short: "Delete a user.",
	Long:  `Remove a user from Drud.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Need to provide a username.")
			os.Exit(1)
		}

		u := &drudapi.User{
			Username: args[0],
		}

		err := drudclient.Get(u)
		if err != nil {
			log.Fatalln(err)
		}

		err = drudclient.Delete(u)
		if err != nil {
			log.Fatalln(err)
		}

	},
}

func init() {
	ListCmd.AddCommand(usersListCmd)
	CreateCmd.AddCommand(usersCreateCmd)
	DeleteCmd.AddCommand(userDeleteCmd)
}
