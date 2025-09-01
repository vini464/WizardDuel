package internal

import (
	"crypto/md5"
	"encoding/hex"
	"sync"

	"github.com/vini464/WizardDuel/tools"
)

func CreateUser(username string, password string, filename string, mu *sync.Mutex) (bool, string) {
	mu.Lock()
	defer mu.Unlock()
	// lendo o arquivo de usurários
	users := make([]tools.UserData, 0)
	GetFileData(filename, &users)
	// A identificação de um usuário ocorre por seu username, por isso não pode haver mais de um usuário com o mesmo username
	for _, user := range users {
		if user.Username == username {
			return false, "Username Already Taken"
		}
	}
	// por segurança, não é legal guardar a senha como texto corrido
	password = hashPassword(password)
	// criando os dados de um novo usuário
	new_user := tools.UserData{Username: username, Password: password, Coins: 0, SavedDecks: nil, MainDeck: tools.Deck{}, AllCards: nil}
	// adicionando o novo usuário no vetor de usuários
	users = append(users, new_user)
	// serializando os dados dos usuários para guardar no users.json
	serialized, err := tools.SerializeJson(users)
	if err != nil {
		return false, "Error While Serializing"
	}
	// escrevendo os dados no arquivo
	bytes, err := OverwriteFile(filename, serialized)
	if err != nil || bytes == 0 { // se a quantidade de bytes escrito for 0 significa que houve algum problema
		return false, "Couldn't Save The File"
	}
	// apenas se o usuário foi criado e salvo com sucesso
	return true, "User Created Successfully"
}

// Procura por um usuário específivo e retorna suas informações e seu indice no slice de usuários (-1) significa que ele não estava no slice
func RetrieveUser(username string, password string, users []tools.UserData) (int, tools.UserData) {
	password = hashPassword(password)
	for index, user := range users {
		if user.Username == username && user.Password == password {
			return index, user
		}
	}
	return -1, tools.UserData{}
}

// pega todos os usuários salvos e retorna em um slice
func RetrieveAllUsers(filename string) []tools.UserData {
	users := make([]tools.UserData, 0)
	GetFileData(filename, &users)
	return users
}

// Atualiza os dados de um usuário
func UpdateUser(usernarme string, password string, new_data tools.UserData, filename string, mu *sync.Mutex) (bool, error) {
	mu.Lock()
	defer mu.Unlock()
	users := RetrieveAllUsers(filename)
	id, _ := RetrieveUser(usernarme, password, users)
	// se encontrou o usuário
	if id != -1 {
		users[id] = new_data // troca os dados antigos pelos atuais
		serialized, err := tools.SerializeJson(users)
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
	return false, nil
}

// dado um username e senha, deleta o usuário indicado (se ele existir)
func DeleteUser(username string, password string, filename string, mu *sync.Mutex) error {
	mu.Lock()
	defer mu.Unlock()
	users := RetrieveAllUsers(filename)
	id, _ := RetrieveUser(username, password, users)
	if id != -1 {
		users = append(users[:id], users[id+1:]...)
		serialized, err := tools.SerializeJson(users)
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

// criptografa a senha para salvar os dados do usuário
func hashPassword(pswd string) string {
	hasher := md5.New()
	hasher.Write([]byte(pswd))
	hash := hex.EncodeToString(hasher.Sum([]byte("strong hash")))
	return hash
}
