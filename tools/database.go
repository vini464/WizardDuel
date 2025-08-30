package tools

import (
	"os"
	"sync"

)

// Esse módulo é responssável pelo controle de arquivos

func readFile[T Serializable](filename string, mu *sync.Mutex) (T, error) {
  mu.Lock()
	file, err := os.ReadFile(filename)
	if err != nil {
		var data T
		mu.Unlock()
		return data, err
	}
	var data T
	err = Deserializejson(file, &data)
	mu.Unlock()
	return data, err
}

// sobreescreve o arquivo indicado com os dados enviados
func overwriteFile(filename string, data []byte, mu *sync.Mutex) (int, error) {
  mu.Lock()
	file, err := os.Create(filename)
	if err != nil {
		mu.Unlock()
		return 0, err
	}
  defer mu.Unlock()
	defer file.Close() // só posso fechar depois de confirmar que não teve erros
	b, err := file.Write(data)
	return b, err
}

// save_user (create)
func CreateUser(credentials UserCredentials, filename string, mu *sync.Mutex) (bool, string) {
	// lendo o arquivo de usurários
	users, err := readFile[[]UserData](filename, mu)
	if err != nil {
		return false, err.Error()
	}

	for _, user := range users {
		if user.Username == credentials.USER {
			return false, "Username Already Taken"
		}
	}

  new_user := UserData{Username: credentials.USER, Password: credentials.PSWD, Coins: 0, SavedDecks: nil}


	users = append(users, new_user)

	serialized, err := SerializeJson(users)
	if err != nil {
		return false, "Error While Serializing"
	}

	b, err := overwriteFile(filename, serialized, mu)
	if err != nil || b == 0 {
		return false, "Couldn't Save The File"
	}
	return true, "User Created Successfully"
}

func DeleteUser(user_info UserCredentials, filename string, mu *sync.Mutex) (bool, error) {
	users, err := readFile[[]UserCredentials](filename, mu)
	if err != nil {
		return false, err
	}
	found := false
	var id int
	for index, user := range users {
		if user.USER == user_info.USER && user.PSWD == user_info.PSWD {
			found = true
			id = index
		}
	}
	if found {
		users = append(users[:id], users[id+1:]...)
		serialized, err := SerializeJson(users)
		if err != nil {
			return false, err
		}
		_, err = overwriteFile(filename, serialized, mu)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func GetUsers(filename string, mu *sync.Mutex) ([]UserData, error) {
	return readFile[[]UserData](filename, mu)
}
