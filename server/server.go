package main

import (
	"fmt"
	"io"
	"net"
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
	var request tools.Message
	err := tools.Deserializejson(income, &request)
	if err != nil {
		fmt.Println("[error] - error while deserializing:", err)
	}
	switch request.CMD {
	case tools.Register.String():
		data, ok := request.DATA.(tools.UserInfo)
		if !ok {
			fmt.Println("[error] - bad request")
			break
		}
		ok, desc := tools.CreateUser(data, "bd/users.json")
		var cmd string
		if ok {
			cmd = "ok"
		} else {
			cmd = "error"
		}
		var response []byte
		for response, err = tools.SerializeMessage(cmd, desc); err != nil; {
		}
		send_channel <- response
	case tools.Login.String():
	case tools.Logout.String():
	case tools.GetBooster.String():
	case tools.Play.String():
	case tools.SaveDeck.String():
	case tools.PlaceCard.String():
	case tools.Surrender.String():
	case tools.DrawCard.String():
	case tools.DiscardCard.String():
	case tools.SkipPhase.String():
	default:
		fmt.Println("[error] - unknown command")
	}
}

func hashPassword(pswd string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pswd), bcrypt.DefaultCost)
	return string(bytes), err
}

