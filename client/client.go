package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"github.com/vini464/WizardDuel/tools"
)

func main() {
	conn, err := net.Dial(tools.SERVER_TYPE, tools.PATH)
	if err != nil {
		fmt.Println("[error] - Unable to connect!", err)
	}

	handleConnection(conn)
}

func handleConnection(conn net.Conn) {
	receive_channel := make(chan []byte)
	send_channel := make(chan []byte)
	error_channel := make(chan error)
	var wg sync.WaitGroup
	defer wg.Wait()

	go tools.ReceiveHandler(conn, receive_channel, &wg, error_channel)
	go tools.SendHandler(conn, send_channel, &wg, error_channel)

	for {
		c := initialPage()
		if c == "0" {
			os.Exit(0)
		}
		var credentials tools.UserCredentials
		credentials.USER = tools.Input("Digite seu username:\n> ")
		credentials.PSWD = hashPassword(tools.Input("Digite sua senha:\n> "))
		var cmd string
		if c == "1" {
			cmd = tools.Login.String()
		} else {
			cmd = tools.Register.String()
		}
		var serialized []byte
		var err error
		for serialized, err = tools.SerializeMessage(cmd, credentials); err != nil; {
		}
		send_channel <- serialized
		rec_bytes := <-receive_channel
		var response tools.Message
		err = tools.Deserializejson(rec_bytes, &response)
		if err != nil {
			fmt.Println("[error] - error while le deserializing message: unknown response", err)
			continue
		}
		switch response.CMD {
		case "ok":
			if c == "1" {
				//fmt.Println("Your Are Logged!")
				online(credentials, send_channel, receive_channel, error_channel)
			} else {
				fmt.Println("User registered successfully")
			}
		case "error":
			if c == "1" {
				fmt.Println("Unable to login:", response.DATA)
			} else {
				fmt.Println("Unable to register:", response.DATA)
			}
		default:
			fmt.Println("Unknown command...", response.CMD)
		}
	}
}

func initialPage() string {
	choice := ""
	for choice != "1" && choice != "2" && choice != "0" {
		fmt.Println("=-=-=-=-=-=-=-=-=-=-=-=-=--=-=--===-=-=-=-=-")
		fmt.Println("=-=-=-=-=-=-=-WizardDuel-=-=--===-=-=-=-=-=-")
		fmt.Println("=-=-=-=-=-=-=-=-=-=-=-=-=--=-=--===-=-=-=-=-")
		fmt.Println("1 - Login")
		fmt.Println("2 - SignIn")
		fmt.Println("0 - Exit")
		choice = tools.Input("> ")
		exec.Command("clear")
	}
	return choice
}

func online(credentials tools.UserCredentials, send_channel chan []byte, receive_channel chan []byte, error_channel chan error) {
	for {
		choice := menu("EXIT", "PLAY")
		if choice == 1 {
			os.Exit(0)
		}
		// iniciando uma partida
		sendRequest(tools.Play.String(), credentials, send_channel)
		var receivedData any
    QUEUE_LOOP:
		for {
			select {
			case serialized := <-receive_channel:
				var receive tools.Message
				err := tools.Deserializejson(serialized, &receive)
				if err != nil {
					fmt.Println("[error] an error occourred", err)
					os.Exit(1)
				}
				switch receive.CMD {
				case "ok":
					fmt.Println("You are playing with: ", receive.DATA) // receive.Data vai ser o GameState
					receivedData = receive.DATA
          match(receivedData, credentials, send_channel)
					break QUEUE_LOOP // só sai do loop quando encontrar uma partida
				case "error":
					fmt.Println("Unable to Play:", receive.DATA)
					break QUEUE_LOOP // só sai do loop quando encontrar uma partida
				case "queued":
					fmt.Println("Waiting for an opponent...")
				default:
					fmt.Println("[error] unknown command:", receive.CMD)
					break QUEUE_LOOP // só sai do loop quando encontrar uma partida
				}
			case err := <-error_channel:
				fmt.Println("[error] an error occourred", err)
			}
		}
		// em uma partida
		fmt.Println("out of the select")
	}
}

func match(receivedData any, credentials tools.UserCredentials, send_channel chan []byte) {
	for {
		fmt.Println("inside game loop")
		exec.Command("clear")
		gamestate := getData[tools.GameState](receivedData)
		fmt.Println("TURN:", gamestate.Turn)
		fmt.Println("PHASE:", gamestate.Phase)

		fmt.Println("Opponent info:")
		fmt.Println("name:", gamestate.Opponent.Username)
		fmt.Println("hand:", gamestate.Opponent.Hand)
		fmt.Println("deck:", gamestate.Opponent.Deck)
		fmt.Println("graveyard:", gamestate.Opponent.Graveyard)
		fmt.Println("HP:", gamestate.Opponent.HP, "SP:", gamestate.Opponent.SP, "Energy:", gamestate.Opponent.Energy, "Crystals:", gamestate.Opponent.Crystals)

		fmt.Println("Your info:")
		fmt.Println("hand:", gamestate.You.Hand)
		fmt.Println("deck:", gamestate.You.Deck)
		fmt.Println("graveyard:", gamestate.You.Graveyard)
		fmt.Println("HP:", gamestate.You.HP, "SP:", gamestate.You.SP, "Energy:", gamestate.You.Energy, "Crystals:", gamestate.You.Crystals)

		if gamestate.Turn == credentials.USER {
			switch gamestate.Phase {
			case tools.Refill.String():
				tools.Input("You want to do something?\n> ")
				sendRequest(tools.SkipPhase.String(), "", send_channel)
			case tools.Draw.String():
				tools.Input("You want to do something?\n> ")
				sendRequest(tools.SkipPhase.String(), "", send_channel)
			case tools.Main.String():
				mainPhase(gamestate.You.Hand, send_channel)
			case tools.Maintenance.String():
				tools.Input("You want to do something?\n> ")
				sendRequest(tools.SkipPhase.String(), "", send_channel)
			case tools.End.String():
				tools.Input("You want to do something?\n> ")
				sendRequest(tools.SkipPhase.String(), "", send_channel)
			default:
			}
		}
	}
}

func menu(last string, args ...string) int {
	maxOpt := len(args)
	if last == "" {
		maxOpt--
	}
	for {
		for id, arg := range args {
			fmt.Println(id, "-", arg)
		}
		if last != "" {
			fmt.Println(len(args), "-", "Back")
		}
		input := tools.Input("Select a number\n> ")
		choice, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("opção inválida!")
		}
		if choice >= 0 && choice <= maxOpt {
			return choice
		}
	}
}

func mainPhase(hand []tools.Card, send_channel chan []byte) {
	for {
		for id, card := range hand {
			fmt.Println(id, "-", card.NAME, "{ cost:", card.COST, "efects:", card.EFFECTS, "}")
		}
		fmt.Println(len(hand), " - skip")
		input := tools.Input("Select a number\n> ")
		choice, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("opção inválida!")
		}
		if choice >= 0 && choice <= len(hand) {
			if choice == len(hand) {
				sendRequest(tools.SkipPhase.String(), "", send_channel)
				return // encerra a função
			}
			sendRequest(tools.PlaceCard.String(), hand[choice].NAME, send_channel)
			return
		} else {
			fmt.Println("opção inválida!")
		}
	}
}

func getData[T tools.Serializable](data any) T {
	mapped, ok := data.(map[string]interface{})
	if !ok {
		fmt.Println("(!ok)[error] - an error occourred...")
		os.Exit(1)
	}
	ser_map, err := tools.SerializeJson(mapped)
	if err != nil {
		fmt.Println("(err)[error] - an error occourred...", err)
		os.Exit(1)
	}
	var structure T
	err = tools.Deserializejson(ser_map, &structure)
	if err != nil {
		fmt.Println("(err)[error] - an error occourred...", err)
		os.Exit(1)
	}
	return structure
}

func sendRequest(cmd string, data any, send_channel chan []byte) {
	response, err := tools.SerializeMessage(cmd, data)
	if err != nil {
		os.Exit(1)
	}
	send_channel <- response
}

func hashPassword(pswd string) string {
	hasher := md5.New()
	hasher.Write([]byte(pswd))
	hash := hex.EncodeToString(hasher.Sum([]byte("testando hash")))
	return hash
}
