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
		sendResponse("error", "Internal Error", send_channel)
		return
	}
	switch request.CMD {
	case tools.Register.String():
		data, ok := request.DATA.(tools.UserInfo)
		if !ok {
			sendResponse("error", "Bad Request", send_channel)
			return
		}
    hash, err := hashPassword(data.PSWD)
    if (err != nil) {
      sendResponse("error", "Internal Error", send_channel)
      return
    }
    data.PSWD = hash // passando a senha por um hash para melhor segurança
		ok, desc := tools.CreateUser(data, "bd/users.json")
		if ok {
			sendResponse("ok", desc, send_channel)
			return
		} else {
			sendResponse("error", desc, send_channel)
			return
		}
	case tools.Login.String():
		data, ok := request.DATA.(tools.UserInfo)
		if !ok {
			sendResponse("error", "Bad Request", send_channel)
			return
		}
    hash, err := hashPassword(data.PSWD)
    if (err != nil) {
      sendResponse("error", "Internal Error", send_channel)
      return
    }
    data.PSWD = hash // passando a senha por um hash para melhor segurança
		_, ok = ONLINE_PLAYERS[data.USER]
		if ok {
			sendResponse("error", "User Already Logged", send_channel)
			return
		}
		users, err := tools.GetUsers("bd/users.json")
		if err != nil {
			sendResponse("error", "Unable to Find User", send_channel)
			return
		}
		for _, user := range users {
			if user.USER == data.USER && user.PSWD == data.PSWD {
				fmt.Println("[debug] - User:", user.USER, "is now logged!")
				userInfo := UserInfo{user.USER, false, "", send_channel}
        ONLINE_PLAYERS[user.USER] = userInfo
        sendResponse("ok", "User Logged In", send_channel)
        return
			}
		}
		sendResponse("error", "Wrong User Or Password", send_channel)
		return

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

func sendResponse(cmd string, data any, send_channel chan []byte) {
	fmt.Println("[error] -", data)
	var response []byte
	var err error
	for response, err = tools.SerializeMessage(cmd, data); err != nil; {
	}
	send_channel <- response
}

func hashPassword(pswd string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pswd), bcrypt.DefaultCost)
	return string(bytes), err
}
