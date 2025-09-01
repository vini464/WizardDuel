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

var USERDATA tools.UserData

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
    serData, _ := tools.SerializeJson(credentials)
		for serialized, err = tools.SerializeMessage(cmd, serData); err != nil; {
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
        var ok bool
        USERDATA, ok = getData[tools.UserData](response.DATA)
        if !ok {
          fmt.Println("[error]")
        }
				online(credentials, send_channel, receive_channel, error_channel)
			} else {
				fmt.Println("User registered successfully")
			}
		case "error":
			if c == "1" {
				fmt.Println("Unable to login:", string(response.DATA))
			} else {
				fmt.Println("Unable to register:", string(response.DATA))
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
		choice := menu("EXIT", "PLAY", "OPEN BOOSTER", "SEE CARDS")
		if choice == 0 {
			os.Exit(0)
		}

		switch choice {
		case 1:
			// iniciando uma partida
			sendRequest(tools.Play.String(), credentials, send_channel)
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
						match(receive.DATA, credentials, send_channel)
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
		case 2:
			sendRequest(tools.GetBooster.String(), "", send_channel)
			select {
			case serialized := <-receive_channel:
				var msg tools.Message
				err := tools.Deserializejson(serialized, &msg)
				if err != nil {
					fmt.Println("[error] an error occourred", err)
					os.Exit(1)
				}
        data, ok := getData[[]tools.Card](msg.DATA)
        if !ok {
          fmt.Println("error")
          return
        }
        if (USERDATA.AllCards == nil) {
          USERDATA.AllCards = make([]tools.Card, 0)
        }
        for _, card := range data {
          fmt.Println(card)
          found := false
          for id, c := range USERDATA.AllCards {
            if card.Name == c.Name {
              USERDATA.AllCards[id].Qnt ++
              found = true
              break
            } 
          }
          if !found {
            USERDATA.AllCards = append(USERDATA.AllCards, card)
          }
        }
			case err := <-error_channel:
				fmt.Println("[error] an error occourred", err)
			}
    case 3: 
        for _, card := range USERDATA.AllCards {
          fmt.Println(card)
      }
      

		}
		// em uma partida
		fmt.Println("out of the select")
	}
}

func match(receivedData []byte, credentials tools.UserCredentials, send_channel chan []byte) {
	for {
		fmt.Println("inside game loop")
		exec.Command("clear")
		gamestate, ok := getData[tools.GameState](receivedData)
    if !ok{
      fmt.Println("erro")
      return
    }
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

func menu(args ...string) int {
	for {
		for id, arg := range args {
			fmt.Println(id, "-", arg)
		}
		input := tools.Input("Select a number\n> ")
		choice, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("opção inválida!")
		}
		if choice >= 0 && choice < len(args) {
			return choice
		}
	}
}

func mainPhase(hand []tools.Card, send_channel chan []byte) {
	for {
		for id, card := range hand {
			fmt.Println(id, "-", card.Name, "{ cost:", card.Cost, "efects:", card.Effects, "}")
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
			sendRequest(tools.PlaceCard.String(), hand[choice].Name, send_channel)
			return
		} else {
			fmt.Println("opção inválida!")
		}
	}
}

func getData[T tools.Serializable](data []byte) (T, bool) {
	var structure T
  err := tools.Deserializejson(data, &structure)
	if err != nil {
		fmt.Println("[error] - an error occourred...", err)
		return structure, false
	}
	return structure, true
}

func sendRequest[T tools.Serializable](cmd string, data T, send_channel chan []byte) {
  serData, _ := tools.SerializeJson(data)
	response, err := tools.SerializeMessage(cmd, serData)
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
