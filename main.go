package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

type wsClients []socketio.Socket

func saveTweet(t *twitter.Tweet) {
	// save tweet to mongodb
}

func printTweet(t *twitter.Tweet) {
	green := color.New(color.FgHiGreen, color.Bold).SprintfFunc()
	// print tweet to cli for development
	fmt.Println(green(t.User.ScreenName), t.Text)
}

func (clients *wsClients) emitTweet(tweet *twitter.Tweet) {
	tweetJSON, _ := json.Marshal(tweet)
	for _, client := range *clients {
		client.Emit("tweet", string(tweetJSON))
	}
}

func (clients *wsClients) startServer() {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	red := color.New(color.FgHiRed, color.Bold).SprintfFunc()
	server.On("connection", func(socket socketio.Socket) {
		*clients = append(*clients, socket)
		log.Println(red("New client added: "), len(*clients))

		socket.On("disconnection", func() {
			newClients := make(wsClients, 0, len(*clients)-1)
			for _, client := range *clients {
				if client.Id() != socket.Id() {
					newClients = append(newClients, client)
				}
			}
			*clients = newClients
			log.Println(red("Client removed: "), len(*clients))
		})
	})

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./client")))
	log.Println(red("Webserver started"))
	go func() {
		log.Fatal(http.ListenAndServe("0.0.0.0:5000", nil))
	}()

}

func main() {
	red := color.New(color.FgHiRed, color.Bold).SprintfFunc()

	ws := new(wsClients)
	ws.startServer()

	twitterKeys := new(keys)
	twitterKeys.get("settings.json")
	config := oauth1.NewConfig(twitterKeys.ConsumerKey, twitterKeys.ConsumerSecret)
	token := oauth1.NewToken(twitterKeys.AccessToken, twitterKeys.AccessSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)
	demux := twitter.NewSwitchDemux()

	demux.Warning = func(warning *twitter.StallWarning) {
		log.Println(warning.Message)
	}

	demux.Tweet = func(tweet *twitter.Tweet) {
		if tweet.RetweetedStatus != nil || tweet.QuotedStatus != nil {
			return
		}
		//go printTweet(tweet)
		go saveTweet(tweet)
		go ws.emitTweet(tweet)
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

	fmt.Println(red("Closing Stream"))
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
