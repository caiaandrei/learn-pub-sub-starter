package main

import (
	"fmt"
	"log"

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
	ch, err := conn.Channel()
	if err != nil {
		fmt.Println(err)
		return
	}
	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Println(err)
		return
	}

	state := gamelogic.NewGameState(userName)

	// subscribe to pause/resume
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+state.GetUsername(),
		routing.PauseKey,
		pubsub.SimpleQueueTransient,
		handlerPause(*state),
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	// subscribe to moves from other players
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+state.GetUsername(),
		routing.ArmyMovesPrefix+".*",
		pubsub.SimpleQueueTransient,
		handlerMove(state, ch),
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	//subscrive to war messages from other players
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix+".*",
		pubsub.SimpleQueueDurable,
		handlerWarMsgs(state),
	)
	if err != nil {
		fmt.Println(err)
		return
	}

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
			outcome, err := state.CommandMove(words)
			if err != nil {
				fmt.Println(err)
				continue
			}
			pubsub.PublishJson(
				ch,
				routing.ExchangePerilTopic,
				routing.ArmyMovesPrefix+"."+state.GetUsername(),
				outcome,
			)
		case "status":
			state.CommandStatus()
		case "help":
			gamelogic.PrintServerHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("Unknown command")
			continue
		}
	}
}

func handlerPause(gs gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, ch *amqp.Channel) func(move gamelogic.ArmyMove) pubsub.AckType {
	return func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		outcome := gs.HandleMove(move)
		fmt.Println(outcome)

		switch outcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJson(
				ch,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+"."+gs.GetUsername(),
				gamelogic.RecognitionOfWar{
					Attacker: move.Player,
					Defender: gs.GetPlayerSnap(),
				},
			)
			if err != nil {
				log.Println(err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		default:
			return pubsub.NackDiscard
		}
	}
}

func handlerWarMsgs(gs *gamelogic.GameState) func(rec gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(rec gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		outcome, _, _ := gs.HandleWar(rec)
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
		case gamelogic.WarOutcomeYouWon:
		case gamelogic.WarOutcomeDraw:
			return pubsub.Ack
		}

		log.Println("unknown war outcome")
		return pubsub.NackDiscard
	}
}
