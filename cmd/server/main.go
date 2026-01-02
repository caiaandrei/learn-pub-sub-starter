package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")

	conStr := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(conStr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()
	fmt.Println("Peril server connected to rabbit")

	ch, err := conn.Channel()
	if err != nil {
		fmt.Println(err)
		return
	}

	err = pubsub.PublishJson(ch, string(routing.ExchangePerilDirect), string(routing.PauseKey), routing.PlayingState{IsPaused: true})

	if err != nil {
		fmt.Println(err)
		return
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Println("Peril server is shutting down")
	os.Exit(0)
}
