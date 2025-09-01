package internal

import (
	"fmt"
	"math/rand/v2"
	"sync"

	"github.com/vini464/WizardDuel/tools"
)

type PlayerInfo struct {
	Username     string
	Send_channel chan []byte
	Data         tools.UserData
	Paried       bool
	MainDeck     tools.Deck
	// iformações disponíveis apenas se pariado
	Gamestate        tools.GameState
	PrivateGameState *PrivateGameState // o mesmo para ambos jogadores
}

type PlayerGameData struct {
	Hand      []tools.Card
	Deck      []tools.Card
	Graveyard []tools.Card
	HP        int
	SP        int
	Energy    int
	Crystals  int
	DamageBonus int
}

type PrivateGameState struct {
	mutex       sync.Mutex
	PlayersData map[string]PlayerGameData // map entre o username e os dados
	Turn        string
	Phase       string
	Round       int
}

// cmd só pode ser desistir, pular fase ou jogar carta
type Action struct {
	Cmd  string
	Card tools.Card
}

// atualiza o estado de jogo a partir de uma ação feita pelo jogador1 retorna os estados de jogo dos dois jogadores
func UpdatePrivateGamestate(player1 *PlayerInfo, player2 *PlayerInfo, action Action, mutex *sync.Mutex) (tools.GameState, tools.GameState) {
	mutex.Lock()
	defer mutex.Unlock()
	gamestate := player1.PrivateGameState
	p2_data := gamestate.PlayersData[player2.Username]
	p1_data := gamestate.PlayersData[player1.Username]
	switch action.Cmd {
	case "surrender":
		player2.Data.Coins += 5 // se um jogador perde o outro ganha 5 moedas (um booster custa 10 moedas)
	case "skip_phase":
		if gamestate.Phase == tools.End.String() {
			gamestate.Turn = player2.Username
			p1_data.DamageBonus = 0
			if len(p1_data.Hand) + len(p1_data.Deck) == 0 {
				p1_data.HP -- 
			}
		}
		gamestate.Phase = tools.NextPhase(gamestate.Phase)
	case "place_card":
		for _, effect := range action.Card.Effects {
			switch effect.Type {
			case "damage":
				damage := effect.Amount + p1_data.DamageBonus
				extra := p2_data.SP - damage 
				p2_data.SP -= damage
				if p2_data.SP < 0 {
					p2_data.SP = 0
				}
				p2_data.HP -= extra
				p1_data.DamageBonus = 0 // acaba o bonus
			case "heal":
				p1_data.HP += effect.Amount
			case "shield":
				p1_data.SP += effect.Amount
			case "energy":
				p1_data.Energy += effect.Amount
			case "draw":
				p1_data.Hand = append(p1_data.Hand, p1_data.Deck[0])
				p1_data.Deck = p1_data.Deck[0:]
			case "discard":
				for range effect.Amount {
					r := rand.IntN(len(p2_data.Hand))
					p2_data.Graveyard = append(p2_data.Graveyard, p2_data.Hand[r])
					p2_data.Hand = append(p2_data.Hand[:r], p2_data.Hand[r+1:]...)
				}
			case "aoe_damage":
				// TODO: futuramente caso adicione outros jogadores em uma partida isso aqui vai funcionar
				extra := p2_data.SP - effect.Amount
				p2_data.SP -= effect.Amount
				if p2_data.SP < 0 {
					p2_data.SP = 0
				}
				p2_data.HP -= extra
			case "next_spell_damage_bonus":
				p1_data.DamageBonus += effect.Amount
			case "destroy_enemy_shield":
				p2_data.SP = 0
			default:
				fmt.Println("[debug] - unknown effect")
			}
		}
	}
	gamestate.PlayersData[player2.Username] = p2_data
	gamestate.PlayersData[player1.Username] = p1_data

	// atribuindo os dados para os gamestates que serão enviados aos jogadores
	p1_gamestate := tools.GameState{}
	p1_gamestate.Opponent.Username = player2.Username
	p1_gamestate.Opponent.Hand = len(p2_data.Hand)
	p1_gamestate.Opponent.Deck = len(p2_data.Deck)
	p1_gamestate.Opponent.Graveyard = p2_data.Graveyard
	p1_gamestate.Opponent.Crystals = p2_data.Crystals
	p1_gamestate.Opponent.SP = p2_data.SP
	p1_gamestate.Opponent.HP = p2_data.HP
	p1_gamestate.Opponent.Energy = p2_data.Energy
	p1_gamestate.Phase = gamestate.Phase
	p1_gamestate.Turn = gamestate.Turn
	p1_gamestate.Round = gamestate.Round
	p1_gamestate.You.Hand = p1_data.Hand
	p1_gamestate.You.Graveyard = p1_data.Graveyard
	p1_gamestate.You.Deck = len(p1_data.Deck)
	p1_gamestate.You.Crystals = p1_data.Crystals
	p1_gamestate.You.SP = p1_data.SP
	p1_gamestate.You.HP = p1_data.HP
	p1_gamestate.You.Energy = p1_data.Energy

	p2_gamestate := tools.GameState{}
	p2_gamestate.Opponent.Username = player1.Username
	p2_gamestate.Opponent.Hand = len(p1_data.Hand)
	p2_gamestate.Opponent.Deck = len(p1_data.Deck)
	p2_gamestate.Opponent.Graveyard = p1_data.Graveyard
	p2_gamestate.Opponent.Crystals = p1_data.Crystals
	p2_gamestate.Opponent.SP = p1_data.SP
	p2_gamestate.Opponent.HP = p1_data.HP
	p2_gamestate.Opponent.Energy = p1_data.Energy
	p2_gamestate.Phase = gamestate.Phase
	p2_gamestate.Turn = gamestate.Turn
	p2_gamestate.Round = gamestate.Round
	p2_gamestate.You.Hand = p2_data.Hand
	p2_gamestate.You.Graveyard = p2_data.Graveyard
	p2_gamestate.You.Deck = len(p2_data.Deck)
	p2_gamestate.You.Crystals = p2_data.Crystals
	p2_gamestate.You.SP = p2_data.SP
	p2_gamestate.You.HP = p2_data.HP
	p2_gamestate.You.Energy = p2_data.Energy

	return p1_gamestate, p2_gamestate
}
