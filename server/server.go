package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"github.com/vini464/WizardDuel/tools"
  "golang.org/x/crypto/bcrypt"
)

type PlayerData struct {
	hp              int
	sp              int
	crystals        int
	avaiable_energy int
	hand            []tools.Card
	deck            []tools.Card
	graveyard       []tools.Card
	phase           tools.TurnPhase
}

type Game struct {
	players  [2]string     // username de cada jogador
	gameData [2]PlayerData // informações do campo de cada jogador
	turn     int
}

type CardSet struct {
	commons   []tools.Card
	uncommons []tools.Card
	rare      []tools.Card
	legendary []tools.Card
}

type UserInfo struct {
	username     string
	paried       bool
	opponent     string // username do oponente
	send_channel chan []byte
}

type DB_user struct {
	user string `json:"user"`
	hash string `json:"hash"`
}

var QUEUE = make([]string, 0)
var ONLINE_PLAYERS = make(map[string]UserInfo)

func main() {

	fmt.Println("[debug] - iniciando o servidor...")
	listener, err := net.Listen(tools.SERVER_TYPE, tools.PATH)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("[error] - unable to connect!")
			continue
		}

		go handleCLient(conn)
	}
}

func handleCLient(conn net.Conn) {
	var wg sync.WaitGroup
	receive_channel := make(chan []byte)
	send_channel := make(chan []byte)
	error_channel := make(chan error)

	defer conn.Close()
	defer wg.Wait()

	wg.Add(1)
	go tools.ReceiveHandler(conn, receive_channel, &wg, error_channel)
	wg.Add(1)
	go tools.SendHandler(conn, send_channel, &wg, error_channel)

LOOP:
	for {
		select {
		case income := <-receive_channel:
			handleReceive(send_channel, income)
		case err := <-error_channel:
			if err == io.EOF {
				fmt.Println("[error] - client forced to quit")
				break LOOP
			}
		}
	}
}

func handleReceive(send_channel chan []byte, income []byte) {
	var request tools.Request
	err := tools.Deserializejson(income, &request)
	if err != nil {
		fmt.Println("[error] - error while deserializing:", err)
	}
	switch request.CMD {
	case tools.Register:
    var data tools.UserInfo
    data, ok := request.DATA.(tools.UserInfo)
    if (!ok) {
      fmt.Println("[error] - bad request")
      break
    }
		file, err := os.Open("users.json")
		for err != nil {
			file, err = os.Create("users.json")
		}

		var users []DB_user
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&users)
		if err != nil {
			fmt.Println("[error] couldn't open users.json file:", err)
			break
		}
    found := false
    for _, user := range users{
      if data.USER == user.user {
        found = true
      }
    }
    if (found) {
      response := tools.Response{CODE: 402, DESCRIPTION: "User already exist"}
      bytes,err := tools.SerializeJson(response)
      if (err != nil){
        fmt.Println("[error]", err)
      }else{
        send_channel <- bytes
      }
    } else {
      
    }
	case tools.Login:
	case tools.Logout:
	case tools.GetBooster:
	case tools.Play:
	case tools.SaveDeck:
	case tools.PlaceCard:
	case tools.Surrender:
	case tools.DrawCard:
	case tools.DiscardCard:
	case tools.SkipPhase:
	default:
		fmt.Println("[error] - unknown command")
	}
}

func hashPassword(pswd string) (string, error) {
  bytes, err := bcrypt.GenerateFromPassword([]byte(pswd), bcrypt.DefaultCost)
  return string(bytes), err
}


func mountResponse(code int, description string, data any) tools.Response {
	var response tools.Response
	response.CODE = code
	response.DESCRIPTION = description
	response.DATA = data
	return response
}
