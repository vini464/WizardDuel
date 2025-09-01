package tools

import (
	"os"
	"sync"
)

// Esse módulo é responssável pelo controle de arquivos

func ReadFile[T Serializable](filename string) (T, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		var data T
		return data, err
	}
	var data T
	err = Deserializejson(file, &data)
	return data, err
}

// sobreescreve o arquivo indicado com os dados enviados
func OverwriteFile(filename string, data []byte) (int, error) {
	file, err := os.Create(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close() // só posso fechar depois de confirmar que não teve erros
	b, err := file.Write(data)
	return b, err
}

// save_user (create)
func CreateUser(credentials UserCredentials, filename string, mu *sync.Mutex) (bool, string) {
  mu.Lock()
  defer mu.Unlock()
	// lendo o arquivo de usurários
	users, err := ReadFile[[]UserData](filename)
	if err != nil {
		//return false, err.Error()
    users = make([]UserData, 0) 

	}

	for _, user := range users {
		if user.Username == credentials.USER {
			return false, "Username Already Taken"
		}
	}

	new_user := UserData{Username: credentials.USER, Password: credentials.PSWD, Coins: 0, SavedDecks: nil, MainDeck: Deck{}, AllCards: nil}

	users = append(users, new_user)

	serialized, err := SerializeJson(users)
	if err != nil {
		return false, "Error While Serializing"
	}

	b, err := OverwriteFile(filename, serialized)
	if err != nil || b == 0 {
		return false, "Couldn't Save The File"
	}
	return true, "User Created Successfully"
}

func DeleteUser(user_info UserCredentials, filename string, mu *sync.Mutex) (bool, error) {
  mu.Lock()
  defer mu.Unlock()
	users, err := ReadFile[[]UserData](filename)
	if err != nil {
		return false, err
	}

	id, found := findUser(user_info, users)

	if found {
		users = append(users[:id], users[id+1:]...)
		serialized, err := SerializeJson(users)
		if err != nil {
			return false, err
		}
		_, err = OverwriteFile(filename, serialized)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func UpdateUser(credentials UserCredentials, new_data UserData, filename string, mu *sync.Mutex) (bool, error) {
  mu.Lock()
  defer mu.Unlock()
	users, err := ReadFile[[]UserData](filename)
	if err != nil {
		return false, err
	}

	id, found := findUser(credentials, users)

	if found {
		users[id] = new_data // troca os dados antigos pelos atuais
		serialized, err := SerializeJson(users)
		if err != nil {
			return false, err
		}
		_, err = OverwriteFile(filename, serialized)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func findUser(credetials UserCredentials, users []UserData) (int, bool) {
	for index, user := range users {
		if user.Username == credetials.USER && user.Password == credetials.PSWD {
			return index, true
		}
	}
	return -1, false
}

func GetUsers(filename string, mu *sync.Mutex) ([]UserData, error) {
  mu.Lock()
  defer mu.Unlock()
	return ReadFile[[]UserData](filename)
}
