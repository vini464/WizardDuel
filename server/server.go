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
	paried       bool
	opponent     string // username do oponente
	send_channel chan []byte
	data         tools.UserData
}

var QUEUE = make([]string, 0)
var ONLINE_PLAYERS = make(map[string]*UserInfo)

func main() {
	var mu sync.Mutex
	var p_mu sync.Mutex

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

		go handleCLient(conn, &mu, &p_mu)
	}
}

func handleCLient(conn net.Conn, mu *sync.Mutex, p_mu *sync.Mutex) {
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

	var username string
LOOP:
	for {
		select {
		case income := <-receive_channel:
			handleReceive(send_channel, income, &username, mu, p_mu)
		case err := <-error_channel:
			if err == io.EOF {
				fmt.Println("[error] - client forced to quit")
				break LOOP
			}
		}
	}
}

func handleReceive(send_channel chan []byte, income []byte, username *string, mu *sync.Mutex, p_mu *sync.Mutex) {
	var request tools.Message
	err := tools.Deserializejson(income, &request)
	if err != nil {
		fmt.Println("[error] - error while deserializing:", err)
		sendResponse("error", "Internal Error", send_channel)
		return
	}
	switch request.CMD {
	case tools.Register.String():
		register(request, send_channel, mu)
	case tools.Login.String():
		login(request, send_channel, mu, username)
	case tools.Logout.String():
		logout(username, send_channel, mu, p_mu)
	case tools.Surrender.String():
		surrender(*username, send_channel, mu, p_mu)
	case tools.PlaceCard.String():
	case tools.DrawCard.String():
	case tools.DiscardCard.String():
	case tools.SkipPhase.String():
	case tools.GetBooster.String():
	case tools.Play.String():
	case tools.SaveDeck.String():
	default:
		fmt.Println("[error] - unknown command")
	}
}

func surrender(username string, send_channel chan []byte, mu *sync.Mutex, p_mu *sync.Mutex) {
	player, ok := ONLINE_PLAYERS[username]
	if ok {
		if player.paried {
			p_mu.Lock()
			opponent, ok := ONLINE_PLAYERS[player.opponent] // eu sei que ele existe, caso contrário o jogador não estaria pariado
			if player.data.Coins > 0 {
				player.data.Coins--
			}
			player.paried = false
			player.opponent = ""
			sendResponse("lose", player.data, send_channel)
			credentials := tools.UserCredentials{USER: player.data.Username, PSWD: player.data.Password}
			for ok, _ := tools.UpdateUser(credentials, player.data, "db/users.json", mu); !ok; {
			}
			if ok {
				opponent.data.Coins += 2
				opponent.paried = false
				opponent.opponent = ""
				sendResponse("win", opponent.data, opponent.send_channel)
        credentials := tools.UserCredentials{USER: opponent.data.Username, PSWD: opponent.data.Password}
        for ok, _ := tools.UpdateUser(credentials, opponent.data, "db/users.json", mu); !ok; {
        }
			}
		}
		sendResponse("error", "Not in Game", send_channel)
		return
	}
	sendResponse("error", "Offline User", send_channel)
}

func logout(username *string, send_channel chan []byte, mu *sync.Mutex, p_mu *sync.Mutex) {
	user, ok := ONLINE_PLAYERS[*username]
	if !ok {
		sendResponse("error", "User Already Offline", send_channel)
		return
	}
	if !user.paried {
		delete(ONLINE_PLAYERS, *username)
		sendResponse("ok", "Logout Successfully", send_channel)
		return
	}
	surrender(*username, send_channel, mu, p_mu)

}

func login(request tools.Message, send_channel chan []byte, mu *sync.Mutex, username *string) {
	credentials, ok := request.DATA.(tools.UserCredentials)
	if !ok {
		sendResponse("error", "Bad Request", send_channel)
		return
	}
	hash, err := hashPassword(credentials.PSWD)
	if err != nil {
		sendResponse("error", "Internal Error", send_channel)
		return
	}
	credentials.PSWD = hash // passando a senha por um hash para melhor segurança
	_, ok = ONLINE_PLAYERS[credentials.USER]
	if ok {
		sendResponse("error", "User Already Logged", send_channel)
		return
	}
	users, err := tools.GetUsers("bd/users.json", mu)
	if err != nil {
		sendResponse("error", "Unable to Find User", send_channel)
		return
	}
	for _, user := range users {
		if user.Username == credentials.USER && user.Password == credentials.PSWD {
			fmt.Println("[debug] - User:", user.Username, "is now logged!")
			userInfo := UserInfo{paried: false, opponent: "", send_channel: send_channel, data: user}
			ONLINE_PLAYERS[user.Username] = &userInfo
			sendResponse("ok", "User Logged In", send_channel)
			*username = credentials.USER
			return
		}
	}
	sendResponse("error", "Wrong User Or Password", send_channel)
}

func register(request tools.Message, send_channel chan []byte, mu *sync.Mutex) {
	data, ok := request.DATA.(tools.UserCredentials)
	if !ok {
		sendResponse("error", "Bad Request", send_channel)
		return
	}
	hash, err := hashPassword(data.PSWD)
	if err != nil {
		sendResponse("error", "Internal Error", send_channel)
		return
	}
	data.PSWD = hash // passando a senha por um hash para melhor segurança
	ok, desc := tools.CreateUser(data, "bd/users.json", mu)
	if ok {
		sendResponse("ok", desc, send_channel)
		return
	}
	sendResponse("error", desc, send_channel)
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
