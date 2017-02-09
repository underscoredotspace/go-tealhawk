package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/fatih/color"
)

type keys struct {
	ConsumerKey    string `json:"consumerKey"`
	ConsumerSecret string `json:"consumerSecret"`
	AccessToken    string `json:"accessToken"`
	AccessSecret   string `json:"accessSecret"`
}

type tweetCounter struct {
	tweets int64
}

// Load keys/secrets
// Validate keys/secrets
// Authenticate with Twitter
// Start new Twitter stream
// Handle tweets via channel

func saveTweet(t *twitter.Tweet) {
	// save tweet to mongodb
}

func emitTweet(t *twitter.Tweet) {
	// emit tweet to web clients via socket
}

func printTweet(t *twitter.Tweet) {
	green := color.New(color.FgHiGreen, color.Bold).SprintfFunc()
	// print tweet to cli for development
	fmt.Println(green(t.User.ScreenName), t.Text)
}

func (c *tweetCounter) uptweet() {
	c.tweets++
}

func main() {
	twitterKeys := new(keys)
	twitterKeys.get("settings.json")

	tweets := make(chan *twitter.Tweet)

	counter := new(tweetCounter)

	go func() {
		for {
			select {
			case tweet := <-tweets:
				go counter.uptweet()
				go printTweet(tweet)
				go saveTweet(tweet)
				go emitTweet(tweet)
			}
		}
	}()

	config := oauth1.NewConfig(twitterKeys.ConsumerKey, twitterKeys.ConsumerSecret)
	token := oauth1.NewToken(twitterKeys.AccessToken, twitterKeys.AccessSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)
	demux := twitter.NewSwitchDemux()

	demux.Tweet = func(tweet *twitter.Tweet) {
		if tweet.RetweetedStatus != nil || tweet.QuotedStatus != nil {
			return
		}
		tweets <- tweet
	}

	filterParams := &twitter.StreamFilterParams{
		Track: []string{"trump"},
	}
	stream, err := client.Streams.Filter(filterParams)
	if err != nil {
		log.Fatal(err)
	}

	// Receive messages until stopped or stream quits
	go demux.HandleChan(stream.Messages)

	// Wait for SIGINT and SIGTERM (HIT CTRL-C)
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-ch)

	red := color.New(color.FgHiRed, color.Bold).SprintfFunc()
	fmt.Println(red("%d tweets processed", counter.tweets))
	stream.Stop()
}

// Temporary arrangement for ease developing
func (k *keys) get(filename string) error {
	configfile, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer configfile.Close()

	filedata := new(bytes.Buffer)
	_, err = filedata.ReadFrom(configfile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(filedata.Bytes(), &k)
	if err != nil {
		return err
	}

	return nil
}
