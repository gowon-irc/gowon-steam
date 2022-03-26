package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/boltdb/bolt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gowon-irc/go-gowon"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	Prefix string `short:"P" long:"prefix" env:"GOWON_PREFIX" default:"." description:"prefix for commands"`
	Broker string `short:"b" long:"broker" env:"GOWON_BROKER" default:"localhost:1883" description:"mqtt broker"`
	APIKey string `short:"k" long:"api-key" env:"GOWON_STEAM_API_KEY" required:"true" description:"steam api key"`
	KVPath string `short:"K" long:"kv-path" env:"GOWON_STEAM_KV_PATH" default:"kv.db" description:"path to kv db"`
}

const (
	moduleName               = "steam"
	mqttConnectRetryInternal = 5
	mqttDisconnectTimeout    = 1000
)

func setUser(kv *bolt.DB, nick, user []byte) error {
	err := kv.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("steam"))
		return b.Put([]byte(nick), []byte(user))
	})
	return err
}

func getUser(kv *bolt.DB, nick []byte) (user []byte, err error) {
	err = kv.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("steam"))
		v := b.Get([]byte(nick))
		user = v
		return nil
	})
	return user, err
}

func parseArgs(msg string) (command, user string) {
	fields := strings.Fields(msg)

	if len(fields) >= 1 {
		command = fields[0]
	}

	if len(fields) >= 2 {
		user = fields[1]
	}

	return command, user
}

func setUserHandler(kv *bolt.DB, nick, user string) (string, error) {
	if user == "" {
		return "Error: username needed", nil
	}

	err := setUser(kv, []byte(nick), []byte(user))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("set %s's user to %s", nick, user), nil
}

func CommandHandler(kv *bolt.DB, nick, user, apiKey string, f func(string, string) (string, error)) (string, error) {
	if user != "" {
		return f(apiKey, user)
	}

	userC, err := getUser(kv, []byte(nick))
	if err != nil {
		return "", err
	}

	if len(userC) == 0 {
		return "Error: username needed", nil
	}

	return f(apiKey, string(userC))
}

func genSteamHandler(apiKey string, kv *bolt.DB) func(m gowon.Message) (string, error) {
	return func(m gowon.Message) (string, error) {
		command, user := parseArgs(m.Args)

		switch command {
		case "s", "set":
			return setUserHandler(kv, m.Nick, user)
		case "r", "recent":
			return CommandHandler(kv, m.Nick, user, apiKey, steamLastGame)
		case "a", "achievement":
			return CommandHandler(kv, m.Nick, user, apiKey, steamLastAchievement)
		}

		return "one of [s]et, [r]ecent or [a]chievements must be passed as a command", nil
	}
}

func defaultPublishHandler(c mqtt.Client, msg mqtt.Message) {
	log.Printf("unexpected message:  %s\n", msg)
}

func onConnectionLostHandler(c mqtt.Client, err error) {
	log.Println("connection to broker lost")
}

func onRecconnectingHandler(c mqtt.Client, opts *mqtt.ClientOptions) {
	log.Println("attempting to reconnect to broker")
}

func onConnectHandler(c mqtt.Client) {
	log.Println("connected to broker")
}

func main() {
	log.Printf("%s starting\n", moduleName)

	opts := Options{}
	if _, err := flags.Parse(&opts); err != nil {
		log.Fatal(err)
	}

	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.AddBroker(fmt.Sprintf("tcp://%s", opts.Broker))
	mqttOpts.SetClientID(fmt.Sprintf("gowon_%s", moduleName))
	mqttOpts.SetConnectRetry(true)
	mqttOpts.SetConnectRetryInterval(mqttConnectRetryInternal * time.Second)
	mqttOpts.SetAutoReconnect(true)

	mqttOpts.DefaultPublishHandler = defaultPublishHandler
	mqttOpts.OnConnectionLost = onConnectionLostHandler
	mqttOpts.OnReconnecting = onRecconnectingHandler
	mqttOpts.OnConnect = onConnectHandler

	kv, err := bolt.Open(opts.KVPath, 0666, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer kv.Close()

	err = kv.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("steam"))
		return err
	})
	if err != nil {
		log.Fatal(err)
	}

	mr := gowon.NewMessageRouter()
	mr.AddCommand("steam", genSteamHandler(opts.APIKey, kv))
	mr.Subscribe(mqttOpts, moduleName)

	log.Print("connecting to broker")

	c := mqtt.NewClient(mqttOpts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	log.Print("connected to broker")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-sigs

	log.Println("signal caught, exiting")
	c.Disconnect(mqttDisconnectTimeout)
	log.Println("shutdown complete")
}
