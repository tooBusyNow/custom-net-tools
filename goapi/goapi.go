package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/go-vk-api/vk"
)

type User struct {
	ID        int32  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	IsClosed  bool   `json:"is_closed"`
	Sex       int    `json:"sex"`
	Domain    string `json:"domain"`
	BDate     string `json:"bdate"`
	Status    string `json:"status"`
	Hometown  string `json:"home_town"`
	Verified  int    `json:"verified"`
}

type Users struct {
	Count int    `json:"count"`
	Items []User `json:"items"`
}

func main() {

	rdr := bufio.NewReader(os.Stdin)
	fmt.Print("Just paste your VK Token: ")
	token, _ := rdr.ReadString('\n')

	client, err := vk.NewClientWithOptions(
		vk.WithToken(token),
	)

	if err != nil {
		fmt.Println("Invalid token, try again")
		os.Exit(1)
	}

	var response Users
	fields := "home_town,schools,status,domain,sex,bdate,country,city,contacts,universities"
	callErr := client.CallMethod("friends.get", vk.RequestParams{
		"fields": fields,
		"count":  2,
		"order":  "random",
	}, &response)

	if callErr != nil {
		fmt.Println("Some error occurred during getting your friends list")
		os.Exit(2)
	}

	fmt.Println(response.Items[0].Status)
}
