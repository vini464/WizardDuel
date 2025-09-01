package internal

import (
	"os"
	"github.com/vini464/WizardDuel/tools"
)

// Esse módulo é responssável pelo controle de arquivos
func ReadFile(filename string) ([]byte, error) {
	file, err := os.ReadFile(filename)
	return file, err
}

// Essa função pega os dados de um arquivo e deserializa para o tipo indicado
func GetFileData[T tools.Serializable](filename string, data *T) error{
	file_bytes, err := ReadFile(filename)
	if err != nil {
		return err
	}
	err = tools.Deserializejson(file_bytes, data)
	return err
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
