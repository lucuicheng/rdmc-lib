package main

import (
	"github.com/deckarep/gosx-notifier"
	"net/http"
	"strings"
	"time"
)

//a slice of string sites that you are interested in watching
var sites []string = []string{
	"http://www.baidu.com",
	"http://www.jianshu.com",
	"http://www.bilibili.com"}

func main() {
	ch := make(chan string)

	for _, s := range sites {
		go pinger(ch, s)
	}

	for {
		select {
		case result := <-ch:
			if strings.HasPrefix(result, "+") {
				s := strings.Trim(result, "+")
				showNotification("Urgent, can't ping website: " + s)
			}
		}
	}
}

func showNotification(message string) {

	note := gosxnotifier.NewNotification(message)
	note.Title = "Site Down"
	note.Sound = gosxnotifier.Default

	note.Push()
}

//Prefixing a site with a + means it's up, while - means it's down
func pinger(ch chan string, site string) {
	for {
		res, err := http.Get(site)

		if err != nil {
			ch <- "-" + site
		} else {
			if res.StatusCode != 200 {
				ch <- "-" + site
			} else {
				ch <- "+" + site
			}
			res.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}
}
