package main

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/vini464/WizardDuel/tools"
)

const (
	USERDB = "database/users.json"
)

/**
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
**/

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
	main_deck    tools.Deck
	gamestate    tools.GameState // só quando estiver pariado
}

var QUEUE = make([]string, 0)
var ONLINE_PLAYERS = make(map[string]*UserInfo)

func main() {
	var mu sync.Mutex
	var p_mu sync.Mutex
	var q_mu sync.Mutex

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

		go handleCLient(conn, &mu, &p_mu, &q_mu)
	}
}

func handleCLient(conn net.Conn, mu *sync.Mutex, p_mu *sync.Mutex, q_mu *sync.Mutex) {
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
			handleReceive(send_channel, income, &username, mu, p_mu, q_mu)
		case err := <-error_channel:
			if err == io.EOF {
				fmt.Println("[error] - client forced to quit")
				var index int
				found := false
				for id, user := range QUEUE {
					if user == username {
						found = true
						index = id
						break
					}
				}
				if found {
					QUEUE = append(QUEUE[:index], QUEUE[index+1:]...) // tiro o cara da fila
				} else {
					if ONLINE_PLAYERS[username].paried {
						surrender(username, send_channel, mu, p_mu)
					}
				}
				delete(ONLINE_PLAYERS, username)
				break LOOP
			}
		}
	}
}

func handleReceive(send_channel chan []byte, income []byte, username *string, mu *sync.Mutex, p_mu *sync.Mutex, q_mu *sync.Mutex) {
	var request tools.Message
	err := tools.Deserializejson(income, &request)
	if err != nil {
		fmt.Println("[error] - error while deserializing:", err)
		sendResponse("error", "Internal Error", send_channel)
		return
	}
	switch request.CMD {
	case tools.Register.String():
		data, ok := getData[tools.UserCredentials](request.DATA)
		if ok {
			register(data, send_channel, mu)
		} else {
			sendResponse("error", "bad request", send_channel)
		}
	case tools.Login.String():
		data, ok := getData[tools.UserCredentials](request.DATA)
		if ok {
			login(data, send_channel, mu, p_mu, username)
		} else {
			sendResponse("error", "bad request", send_channel)
		}
	case tools.Logout.String():
		logout(username, send_channel, mu, p_mu)
	case tools.Surrender.String():
		surrender(*username, send_channel, mu, p_mu)
	case tools.Play.String():
    fmt.Println("play command")
		play(*username, send_channel, q_mu, p_mu)
    fmt.Println("mutext unlocked")
	case tools.PlaceCard.String():
		data, ok := getData[string](request.DATA)
		if ok {
			placeCard(*username, data, send_channel, p_mu)
		} else {
			sendResponse("error", "bad request", send_channel)
		}
	case tools.DrawCard.String():
		sendResponse("error", "not implemented", send_channel)
	case tools.DiscardCard.String():
		sendResponse("error", "not implemented", send_channel)
	case tools.SkipPhase.String():
		sendResponse("error", "not implemented", send_channel)
	case tools.GetBooster.String():
		sendResponse("error", "not implemented", send_channel)
	case tools.SaveDeck.String():
	default:
		fmt.Println("[error] - unknown command")
	}
}

func placeCard(username string, cardname string, send_channel chan []byte, p_mu *sync.Mutex) {
	p_mu.Lock()
	defer p_mu.Unlock()
	if ONLINE_PLAYERS[username].paried {
		hand := ONLINE_PLAYERS[username].gamestate.You.Hand
		cardId := -1
		for id, card := range hand {
			if card.NAME == cardname {
				cardId = id
			}
		}
		if cardId == -1 {
			sendResponse("error", "You dont have that card in hand", send_channel)
			return
		}
		card := hand[cardId]
		if card.COST > ONLINE_PLAYERS[username].gamestate.You.Energy {
			sendResponse("error", "You dont have enougth energy", send_channel)
			return
		}
		for _, effect := range card.EFFECTS {
			switch effect.TYPE {
			case "damage":
				fmt.Println("You dealt", effect.AMOUNT, "damage")
				ONLINE_PLAYERS[username].gamestate.Opponent.HP -= effect.AMOUNT
			case "heal":
				fmt.Println("You heal", effect.AMOUNT, "HP")
				ONLINE_PLAYERS[username].gamestate.You.HP += effect.AMOUNT
			default:
				fmt.Println("Unknown effect")
			}
		}
    // remove a carta da mão
    ONLINE_PLAYERS[username].gamestate.You.Hand = append(ONLINE_PLAYERS[username].gamestate.You.Hand[:cardId], ONLINE_PLAYERS[username].gamestate.You.Hand[cardId:]...)
	}
}

func play(username string, send_channel chan []byte, q_mu *sync.Mutex, p_mu *sync.Mutex) {
	q_mu.Lock()
	p_mu.Lock()
	defer q_mu.Unlock()
	defer p_mu.Unlock()
  
  // o cara não tem um deck
  if ONLINE_PLAYERS[username].main_deck.DeckName == "" && len(ONLINE_PLAYERS[username].main_deck.Cards) == 0  {
    sendResponse("error", "You don't have a deck", send_channel)
    return
  }

	var opponent_name string
	if len(QUEUE) > 0 {
		opponent_name, QUEUE = tools.Dequeue(QUEUE)
		gamestate := &ONLINE_PLAYERS[username].gamestate
		op_gamestate := &ONLINE_PLAYERS[opponent_name].gamestate

    // sempre o jogador que estava esperando começa o jogo
		setGameState(gamestate, username, opponent_name)
		setGameState(op_gamestate, opponent_name, opponent_name)

		setOpponentState(gamestate, op_gamestate, opponent_name)
		setOpponentState(op_gamestate, gamestate, username)

		sendResponse("ok", *gamestate, send_channel)
		sendResponse("ok", *op_gamestate, ONLINE_PLAYERS[opponent_name].send_channel)

	} else {
		QUEUE = tools.Enqueue(QUEUE, username)
	}
}

func setOpponentState(gamestate *tools.GameState, op_gamestate *tools.GameState, op_name string) {
	gamestate.Opponent.Username = op_name
	gamestate.Opponent.Crystals = op_gamestate.Opponent.Crystals
	gamestate.Opponent.Deck = op_gamestate.You.Deck
	gamestate.Opponent.HP = op_gamestate.You.HP
	gamestate.Opponent.SP = op_gamestate.You.SP
	gamestate.Opponent.Energy = op_gamestate.You.Energy
	gamestate.Opponent.Graveyard = op_gamestate.You.Graveyard
	gamestate.Opponent.Hand = len(op_gamestate.You.Hand)

}

func setGameState(gamestate *tools.GameState, username string, turn string) {
	gamestate.You.Crystals = 0
	gamestate.You.HP = 10
	gamestate.You.SP = 10
	gamestate.You.Energy = 0
	gamestate.You.Graveyard = make([]tools.Card, 0)
	gamestate.You.Hand = append(gamestate.You.Hand, ONLINE_PLAYERS[username].main_deck.Cards[:5]...)
	gamestate.You.Deck = len(ONLINE_PLAYERS[username].main_deck.Cards)
	gamestate.Turn = turn
	gamestate.Phase = tools.Refill.String()
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
			for ok, _ := tools.UpdateUser(credentials, player.data, USERDB, mu); !ok; {
			}
			if ok {
				opponent.data.Coins += 2
				opponent.paried = false
				opponent.opponent = ""
				sendResponse("win", opponent.data, opponent.send_channel)
				credentials := tools.UserCredentials{USER: opponent.data.Username, PSWD: opponent.data.Password}
				for ok, _ := tools.UpdateUser(credentials, opponent.data, USERDB, mu); !ok; {
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
	delete(ONLINE_PLAYERS, *username)
	sendResponse("ok", "Logout Successfully", send_channel)
}

func login(credentials tools.UserCredentials, send_channel chan []byte, mu *sync.Mutex, p_mu *sync.Mutex, username *string) {
	p_mu.Lock()
	defer p_mu.Unlock()
	_, ok := ONLINE_PLAYERS[credentials.USER]
	if ok {
		sendResponse("error", "User Already Logged", send_channel)
		return
	}
	users, err := tools.GetUsers(USERDB, mu)
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

func register(credentials tools.UserCredentials, send_channel chan []byte, mu *sync.Mutex) {
	fmt.Println("[debug] - message type:", credentials)
	ok, desc := tools.CreateUser(credentials, USERDB, mu)
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
func getData[T tools.Serializable](data any) (T, bool) {
	var structure T
	mapped, ok := data.(map[string]interface{})
	if !ok {
		fmt.Println("[error] - an error occourred...")
		return structure, false
	}
	ser_map, err := tools.SerializeJson(mapped)
	if err != nil {
		fmt.Println("[error] - an error occourred...", err)
		return structure, false
	}
	err = tools.Deserializejson(ser_map, &structure)
	if err != nil {
		fmt.Println("[error] - an error occourred...", err)
		return structure, false
	}
	return structure, true
}
