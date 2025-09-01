package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"os"
	"sync"

	"github.com/vini464/WizardDuel/tools"
//	"github.com/vini464/WizardDuel/server/internal"
)

const (
	USERDB    = "database/users.json"
	CARDSFILE = "database/cards.json"
)

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
	var c_mu sync.Mutex

	// Verifica a quantidade do stock
	sum := 0
	cards, err := tools.ReadFile[[]tools.Card](CARDSFILE)
	if err != nil {
		fmt.Println("Some shit happen :/", err)
		os.Exit(1)
	} else {
		for _, card := range cards {
			sum += card.Qnt
		}
		if sum < 6000 && sum > 0 {
			updateStock(6000%sum, &c_mu) // atualiza a quantidade de cartas se ela estiver abaixo do mínimo
		} else if sum == 0 {
			updateStock(6000, &c_mu) // atualiza a quantidade de cartas se ela estiver abaixo do mínimo
		}
	}

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

		go handleCLient(conn, &mu, &p_mu, &q_mu, &c_mu)
	}
}

func handleCLient(conn net.Conn, mu *sync.Mutex, p_mu *sync.Mutex, q_mu *sync.Mutex, c_mu *sync.Mutex) {
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
			handleReceive(send_channel, income, &username, mu, p_mu, q_mu, c_mu)
		case err := <-error_channel:
			if err == io.EOF {
				fmt.Println("[debug] - Client Was Disconnected")
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
        mu.Lock()
        credentials := tools.UserCredentials{USER: username, PSWD: ONLINE_PLAYERS[username].data.Password}
        tools.UpdateUser(credentials, ONLINE_PLAYERS[username].data, USERDB, p_mu)
				delete(ONLINE_PLAYERS, username)

        mu.Unlock()
				break LOOP
			}
		}
	}
}

func handleReceive(send_channel chan []byte, income []byte, username *string, mu *sync.Mutex, p_mu *sync.Mutex, q_mu *sync.Mutex, c_mu *sync.Mutex) {
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
			register(data, send_channel, mu, c_mu)
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
	case tools.GetBooster.String():
		booster, err := generateBooster(c_mu)
		if err != nil {
			sendResponse("error", "Internal Error", send_channel)
		}
		// adding cards to player data
		mu.Lock()
		if ONLINE_PLAYERS[*username].data.AllCards == nil {
			ONLINE_PLAYERS[*username].data.AllCards = make([]tools.Card, 0)
		}
		for _, op_card := range booster {
			found := false
			for id, card := range ONLINE_PLAYERS[*username].data.AllCards {
				if op_card.Name == card.Name {
					found = true
					ONLINE_PLAYERS[*username].data.AllCards[id].Qnt++
				}
			}
			if !found {
				ONLINE_PLAYERS[*username].data.AllCards = append(ONLINE_PLAYERS[*username].data.AllCards, op_card)
			}
		}
		defer mu.Unlock()
		sendResponse("ok", booster, send_channel)
	case tools.DrawCard.String():
		sendResponse("error", "not implemented", send_channel)
	case tools.DiscardCard.String():
		sendResponse("error", "not implemented", send_channel)
	case tools.SkipPhase.String():
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
			if card.Name == cardname {
				cardId = id
			}
		}
		if cardId == -1 {
			sendResponse("error", "You dont have that card in hand", send_channel)
			return
		}
		card := hand[cardId]
		if card.Cost > ONLINE_PLAYERS[username].gamestate.You.Energy {
			sendResponse("error", "You dont have enougth energy", send_channel)
			return
		}
		for _, effect := range card.Effects {
			switch effect.Type {
			case "damage":
				fmt.Println("You dealt", effect.Amount, "damage")
				ONLINE_PLAYERS[username].gamestate.Opponent.HP -= effect.Amount
			case "heal":
				fmt.Println("You heal", effect.Amount, "HP")
				ONLINE_PLAYERS[username].gamestate.You.HP += effect.Amount
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
	if ONLINE_PLAYERS[username].main_deck.DeckName == "" && len(ONLINE_PLAYERS[username].main_deck.Cards) == 0 {
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
			*username = credentials.USER
			sendResponse("ok", user, send_channel)
			return
		}

	}
	sendResponse("error", "Wrong User Or Password", send_channel)
}

func register(credentials tools.UserCredentials, send_channel chan []byte, mu *sync.Mutex, c_mu *sync.Mutex) {
	fmt.Println("[debug] - message type:", credentials)
	ok, desc := tools.CreateUser(credentials, USERDB, mu)
	if ok {
		sendResponse("ok", desc, send_channel)
		updateStock(100, c_mu)
		return
	}
	sendResponse("error", desc, send_channel)
}

func sendResponse[T tools.Serializable](cmd string, data T, send_channel chan []byte) {
	fmt.Println("[error] -", data)
	var response []byte
	var err error
  serData , _ := tools.SerializeJson(data)
	for response, err = tools.SerializeMessage(cmd, serData); err != nil; {
	}
	send_channel <- response
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

// operações com as cartas
func updateStock(prints int, mu *sync.Mutex) error {
	mu.Lock()
	defer mu.Unlock()
	cards, err := tools.ReadFile[[]tools.Card](CARDSFILE)
	if err != nil {
		return err
	}
	for id, card := range cards {
		switch card.Rarity {
		case "common":
			card.Qnt += 32 * prints
		case "uncommon":
			card.Qnt += 16 * prints
		case "rare":
			card.Qnt += 8 * prints
		case "legendary":
			card.Qnt += 4 * prints
		default:
			fmt.Println("Unknown type")
		}
		cards[id] = card
	}
	serialized, err := json.MarshalIndent(cards, "", " ")
	if err != nil {
		return err
	}
	_, err = tools.OverwriteFile(CARDSFILE, serialized)
	return err
}

func generateBooster(mu *sync.Mutex) ([]tools.Card, error) {
	mu.Lock()
	defer mu.Unlock()
	cards, err := tools.ReadFile[[]tools.Card](CARDSFILE)
	if err != nil {
		return make([]tools.Card, 0), nil
	}

	booster := make([]tools.Card, 0)

	for len(booster) < 5 {
		rand_id := rand.IntN(len(cards))
		if cards[rand_id].Qnt > 0 {
      card := cards[rand_id]
      card.Qnt = 1 // apenas uma cópia
			booster = append(booster, card)
			cards[rand_id].Qnt--
		}
	}

	serialized, err := json.MarshalIndent(cards, "", " ")
	if err != nil {
		return make([]tools.Card, 0), err
	}
	_, err = tools.OverwriteFile(CARDSFILE, serialized)
	return booster, err
}
