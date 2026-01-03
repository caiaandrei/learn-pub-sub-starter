package main

import (
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
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

	_, queue, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.SimpleQueueDurable)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Queue " + queue.Name + " created!")

	gamelogic.PrintServerHelp()

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "pause":
			fmt.Println("Sending pause message")
			err = pubsub.PublishJson(
				ch,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: true,
				})

			if err != nil {
				fmt.Println(err)
			}
		case "resume":
			fmt.Println("Sending resume message")
			err = pubsub.PublishJson(
				ch,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: false,
				})
			if err != nil {
				fmt.Println(err)
			}
		case "quit":
			fmt.Println("Peril server is shutting down")
			return
		default:
			fmt.Println("Unknown command")
		}
	}
}
