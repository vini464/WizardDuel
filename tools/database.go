package tools

import "os"

// Esse módulo é responssável pelo controle de arquivos

func readFile[T Serializable](filename string) (T, error) {
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
func overwriteFile(filename string, data []byte) (int, error) {
	file, err := os.Create(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close() // só posso fechar depois de confirmar que não teve erros
	b, err := file.Write(data)
	return b, err
}

// save_user (create)
func CreateUser(new_user UserInfo, filename string) (bool, string) {
	// lendo o arquivo de usurários
	users, err := readFile[[]UserInfo](filename)
	if err != nil {
		return false, err.Error()
	}

	for _, user := range users {
		if user.USER == new_user.USER {
			return false, "Username Already Taken"
		}
	}

	users = append(users, new_user)

	serialized, err := SerializeJson(users)
	if err != nil {
		return false, "Error While Serializing"
	}

	b, err := overwriteFile(filename, serialized)
	if err != nil || b == 0 {
		return false, "Couldn't Save The File"
	}
	return true, "User Created Successfully"
}

func DeleteUser(user_info UserInfo, filename string) (bool, error) {
	users, err := readFile[[]UserInfo](filename)
	if err != nil {
		return false, err
	}
	found := false
	var id int
	for index, user := range users {
		if user.USER == user_info.USER {
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
		_, err = overwriteFile(filename, serialized)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func GetUsers(filename string) ([]UserInfo, error) {
  return readFile[[]UserInfo](filename)
}
