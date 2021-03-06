package main

import (
	"bytes"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"encoding/json"

	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/fatih/color"
	"github.com/googollee/go-socket.io"
)

type keys struct {
	ConsumerKey    string `json:"consumerKey"`
	ConsumerSecret string `json:"consumerSecret"`
	AccessToken    string `json:"accessToken"`
	AccessSecret   string `json:"accessSecret"`
}

type ws struct {
	Server *socketio.Server
}

type tweetCounter struct {
	start time.Time
	count float64
}

func main() {
	ws := new(ws)
	if err := ws.start(); err != nil {
		log.Fatalln("Failed to create ws:", err)
	}

	tweets := make(chan *twitter.Tweet, 20)

	err := newTweetStream(tweets)
	if err != nil {
		log.Fatalln("Failed to start Twitter stream:", err)
	}

	tweetCounter := new(tweetCounter)
	go tweetCounter.monitor(5)

	for tweet := range tweets {
		tweet := tweet
		go ws.send(tweet)
		tweetCounter.count++
	}
}

// Start() starts our ws service and serves stream and client
func (ws *ws) start() error {
	server, err := socketio.NewServer(nil)
	if err != nil {
		return err
	}

	server.On("connection", func(socket socketio.Socket) {
		socket.Join("tweets")
	})

	ws.Server = server

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./client")))

	go func() {
		log.Fatal(http.ListenAndServe("0.0.0.0:5000", nil))
	}()

	red := color.New(color.FgHiRed, color.Bold).SprintfFunc()
	log.Println(red("Webserver started"))

	return nil
}

// Send() broadcasts new tweet to connected clients
func (ws *ws) send(tweet *twitter.Tweet) error {
	tweetJSON, err := json.Marshal(tweet)
	if err != nil {
		return err
	}
	ws.Server.BroadcastTo("tweets", "tweet", string(tweetJSON))
	return nil
}

// newTweetStream() starts new Twitter stream and returns channel for new tweets
func newTweetStream(tweets chan *twitter.Tweet) (err error) {
	keys := new(keys)
	err = keys.get()
	if err != nil {
		return
	}

	config := oauth1.NewConfig(keys.ConsumerKey, keys.ConsumerSecret)
	token := oauth1.NewToken(keys.AccessToken, keys.AccessSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)
	demux := twitter.NewSwitchDemux()

	demux.Tweet = func(tweet *twitter.Tweet) {
		if tweet.RetweetedStatus != nil || tweet.QuotedStatus != nil {
			return
		}
		tweets <- tweet
	}

	demux.Warning = func(warning *twitter.StallWarning) {
		log.Println(warning.Message)
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

	red := color.New(color.FgHiRed, color.Bold).SprintfFunc()
	log.Println(red("Twitter stream started"))

	go func() {
		// Wait for SIGINT and SIGTERM (HIT CTRL-C)
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		log.Println(red("Twitter stream stopping..."))
		stream.Stop()
		os.Exit(0)
	}()
	return
}

//
func (tc *tweetCounter) monitor(interval time.Duration) {
	for {
		tc.start = time.Now()
		tc.count = 0
		time.Sleep(interval * time.Second)
		seconds := float64(time.Since(tc.start) / 1000000000)
		log.Printf("%.1f t/s\n", toFixed(tc.count/seconds, 1))
	}
}

// Temporary arrangement to ease development
func (k *keys) get() error {
	configfile, err := os.Open("settings.json")
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

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}
