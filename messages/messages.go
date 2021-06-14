package messages

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Message struct {
	Rate int
	Type int
	Text string
}

var Messages []Message
var MaxRate int = 0
var EmptyMessage Message

var watcher *fsnotify.Watcher
var messageLock sync.RWMutex

func WatchMessages(filename string) {
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer UnwatchMessages()

	fmt.Println("start watching messages file")
	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					fmt.Println("modified file:", event.Name)
					ReadMessages(filename)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Println("watch error:", err)
			}
		}
	}()

	err = watcher.Add(filename)
	if err != nil {
		return
	}
	<-done
}

func UnwatchMessages() {
	watcher.Close()
	fmt.Println("stop watching messages file")
}

func ReadMessages(filename string) {
	messageLock.Lock()
	defer messageLock.Unlock()

	Messages = make([]Message, 0, 4096)
	MaxRate = 0

	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	n := 0
	m := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		b := strings.TrimSpace(scanner.Text())
		if b == "" || strings.HasPrefix(b, ";") {
			continue
		}
		if strings.HasPrefix(b, "-") {
			spaceIndex := strings.Index(b, " ")
			var err error
			if spaceIndex == -1 {
				m, err = strconv.Atoi(b[1:])
			} else {
				m, err = strconv.Atoi(b[1:spaceIndex])
			}
			if err != nil {
				m = 0
			}
			continue
		}
		n += m
		Messages = append(Messages, Message{
			Rate: n,
			Type: m,
			Text: b,
		})
	}
	MaxRate = n
	fmt.Println(len(Messages))
	fmt.Println(MaxRate)
}

func RandomMessage() Message {
	messageLock.RLock()
	defer messageLock.RUnlock()

	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(MaxRate)
	count := len(Messages)

	for i := 0; i < count; i++ {
		message := Messages[i]
		if n < message.Rate {
			return message
		}
	}

	return EmptyMessage
}
