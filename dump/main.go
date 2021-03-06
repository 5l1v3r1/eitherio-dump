package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const MinBackoff = time.Millisecond * 10
const MaxBackoff = time.Second * 10
const WriteLen = 100

type Question struct {
	Option1  string `json:"option_1"`
	Option2  string `json:"option_2"`
	MoreInfo string `json:"moreinfo"`

	Count1 string `json:"option1_total"`
	Count2 string `json:"option2_total"`

	CreatorEmail string `json:"email"`
	CreatorName  string `json:"display_name"`
}

func (q Question) Hash() string {
	return q.Option1 + "\n" + q.Option2
}

type QuestionQuery struct {
	Questions []Question `json:"questions"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: eitherio-dump <output.json>")
		os.Exit(1)
	}

	var questionsLock sync.Mutex
	seenQuestions := map[string]bool{}
	questions := make([]Question, 0)

	var errorCount uint32
	var collisionCount uint32

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		questionsLock.Lock()
		fmt.Println("Saving", len(questions), "results with", atomic.LoadUint32(&errorCount),
			"errors and", atomic.LoadUint32(&collisionCount), "collisions...")
		flushQuestions(questions)
		os.Exit(0)
	}()

	backoff := MinBackoff
	for {
		time.Sleep(backoff)
		if query, err := makeQuery(); err != nil {
			atomic.AddUint32(&errorCount, 1)
			backoff *= 2
			if backoff > MaxBackoff {
				backoff = MaxBackoff
			}
		} else {
			backoff = MinBackoff
			questionsLock.Lock()
			for _, question := range query.Questions {
				if !seenQuestions[question.Hash()] {
					seenQuestions[question.Hash()] = true
					questions = append(questions, question)
					if 0 == len(questions)%WriteLen {
						flushQuestions(questions)
						fmt.Println("Flushing", len(questions), "results with",
							atomic.LoadUint32(&errorCount), "errors and",
							atomic.LoadUint32(&collisionCount), " collisions...")
					}
				} else {
					atomic.AddUint32(&collisionCount, 1)
				}
			}
			questionsLock.Unlock()
		}
	}
}

func makeQuery() (*QuestionQuery, error) {
	resp, err := http.Get("http://either.io/questions/next/100")
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var decoded QuestionQuery
	if err := json.Unmarshal(contents, &decoded); err != nil {
		return nil, err
	}
	return &decoded, nil
}

func flushQuestions(q []Question) {
	data, _ := json.Marshal(q)
	ioutil.WriteFile(os.Args[1], data, 0755)
}
