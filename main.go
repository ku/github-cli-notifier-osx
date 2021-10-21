package main

import (
	"encoding/json"
	"fmt"
	gosxnotifier "github.com/deckarep/gosx-notifier"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Subject struct {
	Title string `json:"title"`
	Url   string `json:"url"`
}

type Repository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

type Notification struct {
	Unread     bool       `json:"unread"`
	Reason     string     `json:"reason"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Subject    Subject    `json:"subject"`
	Url        string     `json:"url"`
	Repository Repository `json:"repository"`
}

func main() {
	dirname, _ := os.UserHomeDir()
	lastNotifFile := fmt.Sprintf("%s/.mcgithubnotif", dirname)

	ticker := time.NewTicker(60 * time.Second)

	for {
		select {
		case <-ticker.C:
			err := notifiyIfNeeded(lastNotifFile)
			if err != nil {
				log.Println(err.Error())
			}
		}
	}
}

func notifiyIfNeeded(filename string) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.github.com/notifications", nil)
	if err != nil {
		return fmt.Errorf("failed to create GET request %s", err.Error())
	}
	token, err := getGithubToken()
	if err != nil {
		return fmt.Errorf("set GITHUB_TOKEN env var or put ~/.github_token file. %s", err.Error())
	}
	req.SetBasicAuth("ku", token)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create GET request %s", err.Error())
	}

	var notifications []Notification
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&notifications)

	if err != nil {
		return fmt.Errorf("failed to create GET request %s", err.Error())
	}

	latest, err := getTimeOfLatestNotification(filename)
	if err != nil {
		return fmt.Errorf("failed to get last notification %s", err.Error())
	}

	if latest.IsZero() {
		// ignore all notifications and notify new ones from now.
	} else {
		for _, n := range notifications {
			if n.UpdatedAt.After(latest) {
				if strings.HasPrefix(n.Repository.FullName, os.Getenv("GITHUB_NOTIFIER_FILTER")) {
					title := fmt.Sprintf("%s %s", n.Reason, n.Subject.Title)
					note := gosxnotifier.NewNotification(title)
					note.Subtitle = n.Repository.Name
					note.Link = n.Subject.Url
					note.Sound = gosxnotifier.Default
					note.Push()
					latest = n.UpdatedAt
					println(title)
				}
			}

			touch(filename)
		}
	}
	return nil
}

func touch(filename string) error {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		f, err := os.Create(filename)
		if err != nil {
			return err
		}
		return f.Close()
	} else {
		now := time.Now()
		return os.Chtimes(filename, now, now)
	}
}
func getTimeOfLatestNotification(filename string) (time.Time, error) {
	t := time.Unix(0, 0)
	info, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return t, nil
		}
		return t, err
	}
	return info.ModTime(), nil
}

func getGithubToken() (string, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if len(token) > 0 {
		return token, nil
	}

	dirname, _ := os.UserHomeDir()
	tokenFile := fmt.Sprintf("%s/.github_token", dirname)
	bytes, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
