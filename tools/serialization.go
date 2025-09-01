package tools

import (
  "encoding/json"
)

func SerializeJson[T Serializable](data T) ([]byte, error) {
  serialized_data, err := json.MarshalIndent(data, "", " ")
  if (err != nil) {
    return make([]byte, 0), err
  }
  return serialized_data, nil
}

func Deserializejson[T Serializable](serialized []byte, data *T) error {
  err := json.Unmarshal(serialized, data)
  return err
}

