package main

import (
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")

	conStr := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(conStr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Println(err)
		return
	}

	_, queue, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+userName,
		routing.PauseKey,
		pubsub.SimpleQueueTransient)

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Queue %v declared and bound!\n", queue.Name)

	state := gamelogic.NewGameState(userName)

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "spawn":
			err = state.CommandSpawn(words)
			if err != nil {
				fmt.Println(err)
				continue
			}
		case "move":
			move, err := state.CommandMove(words)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Printf("Moved to location %s\n", move.ToLocation)
		case "status":
			state.CommandStatus()
		case "help":
			gamelogic.PrintServerHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quite":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("Unknown command")
			continue
		}
	}
}
