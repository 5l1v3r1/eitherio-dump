package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Question struct {
	Option1  string `json:"option_1"`
	Option2  string `json:"option_2"`
	MoreInfo string `json:"moreinfo"`

	Count1 string `json:"option1_total"`
	Count2 string `json:"option2_total"`

	CreatorEmail string `json:"email"`
	CreatorName  string `json:"display_name"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: emails.go <dump.json>")
		os.Exit(1)
	}

	contents, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	var questions []Question
	json.Unmarshal(contents, &questions)

	emails := map[string]string{}
	for _, question := range questions {
		emails[question.CreatorEmail] = question.CreatorName
	}
	for email, name := range emails {
		fmt.Println(email, "-", name)
	}
}
