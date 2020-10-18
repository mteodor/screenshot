package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kbinani/screenshot"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	username = "7c017935-371d-4062-b0d6-f0db6862ecc0"
	password = "84497dc5-c29f-49ff-9362-f360543d5f76"
	channel  = "3ad00857-6fcc-472f-b331-a40a6f988055"
)

func main() {
	var err error

	errs := make(chan error, 2)

	go makeScreenshot()
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	fmt.Println(err.Error())
}

func makeScreenshot() {
	client, _ := connectToMQTTBroker(username, password)
	var b bytes.Buffer
	imgBuf := bufio.NewWriter(&b)
	for _ = range time.Tick(60 * time.Millisecond) {
		bounds := screenshot.GetDisplayBounds(0)

		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			panic(err)
		}

		png.Encode(imgBuf, img)
		imgBuf.Flush()
		dv := base64.StdEncoding.EncodeToString(b.Bytes())

		go func() {
			token := client.Publish("channels/"+channel+"/messages/images", 0, false, "[{\"bn\":\"image\", \"vs\":\""+dv+"\"}]")
			token.Wait()
			fmt.Println("published")
			if token.Error() != nil {
				fmt.Println(token.Error())
			}
		}()

	}
}

func connectToMQTTBroker(username, password string) (mqtt.Client, error) {
	name := fmt.Sprintf("image-%s", username)
	conn := func(client mqtt.Client) {
		fmt.Println(fmt.Sprintf("Client %s connected", name))
	}

	lost := func(client mqtt.Client, err error) {
		fmt.Println(fmt.Sprintf("Client %s disconnected", name))
	}

	opts := mqtt.NewClientOptions().
		AddBroker("tcp://localhost:1883").
		SetClientID(name).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetOnConnectHandler(conn).
		SetConnectionLostHandler(lost)

	if username != "" && password != "" {
		opts.SetUsername(username)
		opts.SetPassword(password)
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()

	if token.Error() != nil {
		return nil, token.Error()
	}
	return client, nil
}
