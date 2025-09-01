package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"os"
	"sync"
	"time"

	"github.com/vini464/WizardDuel/server/internal"
	"github.com/vini464/WizardDuel/tools"
)

const (
	USERDB    = "database/users.json"
	CARDSFILE = "database/cards.json"
)

var QUEUE = make([]*internal.PlayerGameData, 0)
var ONLINE_PLAYERS = make(map[string]*internal.PlayerInfo)

func main() {
	var mu sync.Mutex
	var user_mu sync.Mutex
	var queue_mu sync.Mutex
	var card_mu sync.Mutex

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
			updateStock(6000%sum, &card_mu) // atualiza a quantidade de cartas se ela estiver abaixo do mínimo
		} else if sum == 0 {
			updateStock(6000, &card_mu) // atualiza a quantidade de cartas se ela estiver abaixo do mínimo
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

		go handleCLient(conn, &mu, &user_mu, &queue_mu, &card_mu)
	}
}

func handleCLient(conn net.Conn, mu *sync.Mutex, user_mu *sync.Mutex, q_mu *sync.Mutex, c_mu *sync.Mutex) {
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
			handleReceive(send_channel, income, &username, mu, user_mu, q_mu, c_mu)
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
					if ONLINE_PLAYERS[username].Paried {
						surrender(username, send_channel, mu, user_mu)
					}
				}
				mu.Lock()
				internal.UpdateUser(username, ONLINE_PLAYERS[username].Data.Password, ONLINE_PLAYERS[username].Data, USERDB, user_mu)
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
		play(*username, send_channel, q_mu, p_mu)
		fmt.Println("mutext unlocked")
	case tools.GetBooster.String():
		booster, err := generateBooster(c_mu)
		if err != nil {
			sendResponse("error", "Internal Error", send_channel)
		}
		// adding cards to player data
		mu.Lock()
		if ONLINE_PLAYERS[*username].Data.AllCards == nil {
			ONLINE_PLAYERS[*username].Data.AllCards = make([]tools.Card, 0)
		}
		for _, op_card := range booster {
			found := false
			for id, card := range ONLINE_PLAYERS[*username].Data.AllCards {
				if op_card.Name == card.Name {
					found = true
					ONLINE_PLAYERS[*username].Data.AllCards[id].Qnt++
				}
			}
			if !found {
				ONLINE_PLAYERS[*username].Data.AllCards = append(ONLINE_PLAYERS[*username].Data.AllCards, op_card)
			}
		}
		defer mu.Unlock()
		sendResponse("ok", booster, send_channel)
	case tools.SaveDeck.String():
	default:
		fmt.Println("[error] - unknown command")
	}
}

func play(username string, send_channel chan []byte, q_mu *sync.Mutex, p_mu *sync.Mutex) {
	q_mu.Lock()
	p_mu.Lock()
	defer q_mu.Unlock()
	defer p_mu.Unlock()

	// o cara não tem um deck
	player := ONLINE_PLAYERS[username]
	if player.Data.MainDeck.DeckName == "" && len(player.Data.MainDeck.Cards) == 0 {
		sendResponse("error", "You don't have a deck", send_channel)
		return
	}
	p_data := internal.PlayerGameData{}
	p_data.Username = username
	p_data.Deck = shuffle(player.Data.MainDeck.Cards)
	p_data.Hand = p_data.Deck[:5]
	p_data.Graveyard = make([]tools.Card, 0)
	p_data.Crystals = 0
	p_data.HP = 10
	p_data.SP = 10
	p_data.Energy = 0
	p_data.DamageBonus = 0

	var opponent *internal.PlayerGameData
	if len(QUEUE) > 0 {
		opponent, QUEUE = tools.Dequeue(QUEUE)
		var in_game_mutex sync.Mutex

		// sempre o jogador que estava esperando começa o jogo
		players_data := map[string]internal.PlayerGameData{opponent.Username: *opponent, username: p_data}
		privateGamestate := internal.PrivateGameState{Mutex: &in_game_mutex, Phase: tools.Set.String(), Round: 0, Turn: opponent.Username, PlayersData: players_data}

		op_gamestate, self_gamestate := internal.UpdatePublicGamestate(&privateGamestate)

		sendResponse("ok", self_gamestate, send_channel)
		sendResponse("ok", op_gamestate, ONLINE_PLAYERS[opponent.Username].Send_channel)

	} else {
		QUEUE = tools.Enqueue(QUEUE, &p_data)
	}
}

func surrender(username string, send_channel chan []byte, mu *sync.Mutex, p_mu *sync.Mutex) {
	player, ok := ONLINE_PLAYERS[username]
	if ok {
		if player.Paried {
			p_mu.Lock()
			opponent, ok := ONLINE_PLAYERS[player.Gamestate.Opponent.Username] // eu sei que ele existe, caso contrário o jogador não estaria pariado
			if player.Data.Coins > 0 {
				player.Data.Coins--
			}
			player.Paried = false
			sendResponse("lose", player.Data, send_channel)
			for ok, _ := internal.UpdateUser(username, player.Data.Password, player.Data, USERDB, mu); !ok; {
			}
			if ok {
				opponent.Data.Coins += 2
				opponent.Paried = false
				sendResponse("win", opponent.Data, opponent.Send_channel)
				credentials := tools.UserCredentials{USER: opponent.Data.Username, PSWD: opponent.Data.Password}
				for ok, _ := tools.UpdateUser(credentials, opponent.Data, USERDB, mu); !ok; {
				}
			}
		} else {
			sendResponse("error", "Not in Game", send_channel)
			return
		}
	}
	sendResponse("error", "Offline User", send_channel)
}

func logout(username *string, send_channel chan []byte, mu *sync.Mutex, p_mu *sync.Mutex) {
	user, ok := ONLINE_PLAYERS[*username]
	if !ok {
		sendResponse("error", "User Already Offline", send_channel)
		return
	}
	if !user.Paried {
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
	users, err := tools.GetUsers(USERDB, mu)
	if err != nil {
		sendResponse("error", "Unable to Find User", send_channel)
		return
	}
	for _, user := range users {
		if user.Username == credentials.USER && user.Password == credentials.PSWD {
			fmt.Println("[debug] - User:", user.Username, "is now logged!")
			userInfo := internal.PlayerInfo{Username: *username, Paried: false, Send_channel: send_channel, Data: user}
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
	serData, _ := tools.SerializeJson(data)
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

func shuffle(deck []tools.Card) []tools.Card {
	perm := rand.Perm(len(deck))
	for i := range deck {
		j := perm[i]
		deck[i], deck[j] = deck[j], deck[i]
	}
	return deck
}
