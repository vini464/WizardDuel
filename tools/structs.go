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
	DATA any `json:"data.omitempty"`
}

// Algumas estruturas usadas para enviar mensagens
type UserInfo struct {
	USER string `json:"user"`
	PSWD string `json:"pswd"`
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

type Serializable interface {
	Message | UserInfo | Card | Effect | []UserInfo
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

// example:
/**
{
  CMD: "register",
  DATA: {
    user: "user",
    pswd: "passs"
}
}
**/
