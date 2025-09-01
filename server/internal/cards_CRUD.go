package internal

import (
	"sync"
	"github.com/vini464/WizardDuel/tools"
)

// cria uma carta nova
func CreateCard(card_data tools.Card, filename string, mu *sync.Mutex) bool {
	mu.Lock()
	defer mu.Unlock()
	// lendo o arquivo de cartas
	cards := make([]tools.Card, 0)
	GetFileData(filename, &cards)
	// A identificação de uma carta ocorre por seu nome, por isso não pode haver mais de uma carta com o mesmo nome
	for _, card := range cards {
		if card.Name == card_data.Name {
			return false
		}
	}
	// adicionando a nova carta no vetor de cartas
	cards = append(cards, card_data)
	// serializando os dados das cartas para guardar
	serialized, err := tools.SerializeJson(cards)
	if err != nil {
		return false
	}
	// escrevendo os dados no arquivo
	bytes, err := OverwriteFile(filename, serialized)
	if err != nil || bytes == 0 { // se a quantidade de bytes escrito for 0 significa que houve algum problema
		return false
	}
	// apenas se a carta foi criada e salva com sucesso
	return true
}

// Procura por uma carta específivo e retorna suas informações e seu indice no slice de cartas [(-1) significa que ele não estava no slice]
func RetrieveCard(card_name string, cards []tools.Card) (int, tools.Card) {
	for index, card := range cards {
		if card.Name == card_name{
			return index, card
		}
	}
	return -1, tools.Card{}
}

// pega todos as cartas salvos e retorna em um slice
func RetrieveAllCards(filename string) []tools.Card {
	cards := make([]tools.Card, 0)
	GetFileData(filename, &cards)
	return cards
}

// pega todos as cartas com a mesma raridade e retorna em um slice
func RetrieveSameRarityCards(filename string, rarity string) []tools.Card {
	cards := make([]tools.Card, 0)
	same_rarity := make([]tools.Card, 0)
	GetFileData(filename, &cards)
	for _, card := range cards {
		if (card.Rarity == rarity) {
			same_rarity = append(same_rarity, card)
		}
	}
	return same_rarity
}

// Atualiza os dados de uma carta
func UpdateCard(card_name string, new_data tools.Card, filename string, mu *sync.Mutex) (bool, error) {
	mu.Lock()
	defer mu.Unlock()
	cards := RetrieveAllCards(filename)
	id, _ := RetrieveCard(card_name, cards)
	// se encontrou a carta
	if id != -1 {
		cards[id] = new_data // troca os dados antigos pelos atuais
		serialized, err := tools.SerializeJson(cards)
		if err != nil {
			return false, err
		}
		// escreve os dados no arquivo
		_, err = OverwriteFile(filename, serialized)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil // não encontrou a carta mas não houve erros
}

// dado o nome de uma cata, deleta a carta indicada (se ela existir)
func DeleteCard(card_name string, filename string, mu *sync.Mutex) error {
	mu.Lock()
	defer mu.Unlock()
	cards := RetrieveAllCards(filename)
	id, _ := RetrieveCard(card_name, cards)
	if id != -1 {
		cards = append(cards[:id], cards[id+1:]...)
		serialized, err := tools.SerializeJson(cards)
		if err != nil {
			return err
		}
		_, err = OverwriteFile(filename, serialized)
		if err != nil {
			return err
		}
	}
	return nil
}
