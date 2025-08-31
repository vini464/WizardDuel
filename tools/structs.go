package tools

type TurnPhase int
type Cmd int

const (
	Refill TurnPhase = iota
	Draw
	Main
	Maintenance
	End
	StandBy
	Register Cmd = iota
	Login
	Logout
	GetBooster
	Play
	SaveDeck
	PlaceCard
	Surrender
	SkipPhase
	DrawCard
	DiscardCard
)

var PhaseName = map[TurnPhase]string{
	Refill:      "refill",
	Draw:        "draw",
	Main:        "main",
	Maintenance: "maintenance",
	End:         "end",
	StandBy:     "standby",
}
var CmdName = map[Cmd]string{
	Register:    "register",
	Login:       "login",
	Logout:      "logout",
	GetBooster:  "get_booster",
	Play:        "play",
	SaveDeck:    "save_deck",
	PlaceCard:   "place_card",
	Surrender:   "surrender",
	SkipPhase:   "skip_phase",
	DrawCard:    "draw_card",
	DiscardCard: "discard_card",
}

func (tp TurnPhase) String() string {
	return PhaseName[tp]
}
func (cmd Cmd) String() string {
	return CmdName[cmd]
}

type Message struct {
	CMD  string `json:"cmd"`
	DATA any    `json:"data.omitempty"`
}

type UserCredentials struct {
	USER string `json:"user"`
	PSWD string `json:"pswd"`
}

type Deck struct {
	DeckName string `json:"deckname"`
	Cards    []Card `json:"cards"`
}

type UserData struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	Coins      int    `json:"coins"`
	SavedDecks []Deck `json:"savedDecks"`
}

type Effect struct {
	TYPE   string `json:"type"`
	AMOUNT int    `json:"amount"`
}

type Card struct {
	NAME    string   `json:"name"`
	COST    int      `json:"cost"`
	EFFECTS []Effect `json:"effects"`
}

type GameState struct {
	Opponent struct {
		Hand      int    `json:"hand"`
		Deck      int    `json:"deck"`
		Graveyard []Card `json:"graveyard"`
		HP        int    `json:"hp"`
		SP        int    `json:"sp"`
		Energy    int    `json:"energy"`
		Crystals  int    `json:"crystals"`
	} `json:"opponent"`
	You struct {
		Hand      []Card `json:"hand"`
		Deck      int    `json:"deck"`
		Graveyard []Card `json:"graveyard"`
		HP        int    `json:"hp"`
		SP        int    `json:"sp"`
		Energy    int    `json:"energy"`
		Crystals  int    `json:"crystals"`
	} `json:"you"`
	Turn  string `json:"turn"`
	Phase string `json:"phase"`
}

type Serializable interface {
	GameState | Message | UserCredentials | Card | Effect | []UserCredentials | UserData | []UserData | map[string]string | map[string]interface{}
}

func NextPhase(actualPhase TurnPhase) TurnPhase {
	switch actualPhase {
	case StandBy:
		return Refill
	case Refill:
		return Draw
	case Draw:
		return Main
	case Main:
		return Maintenance
	case Maintenance:
		return End
	case End:
		return StandBy
	default:
		return actualPhase
	}
}

func CreateMessage(cmd string, data any) Message {
	messase := Message{CMD: cmd, DATA: data}
	return messase
}

func SerializeMessage(cmd string, data any) ([]byte, error) {
	message := CreateMessage(cmd, data)
	serialzed, err := SerializeJson(message)
	return serialzed, err
}
