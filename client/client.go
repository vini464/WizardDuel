package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"

	"github.com/vini464/WizardDuel/tools"
)

func main() {
	conn, err := net.Dial(tools.SERVER_TYPE, tools.PATH)
	if err != nil {
		fmt.Println("[error] - Unable to connect!", err)
	}

	handleConnection(conn)
}

func handleConnection(conn net.Conn) {
	receive_channel := make(chan []byte)
	send_channel := make(chan []byte)
	error_channel := make(chan error)
	var wg sync.WaitGroup
	defer wg.Wait()

	go tools.ReceiveHandler(conn, receive_channel, &wg, error_channel)
	go tools.SendHandler(conn, send_channel, &wg, error_channel)

	for {
		c := initialPage()
		if c == "0" {
			os.Exit(0)
		}
		var credentials tools.UserCredentials
		credentials.USER = tools.Input("Digite seu username:\n> ")
		credentials.PSWD = tools.Input("Digite sua senha:\n> ")
		var cmd string
		if c == "1" {
			cmd = tools.Login.String()
		} else {
			cmd = tools.Register.String()
		}
		var serialized []byte
		var err error
		for serialized, err = tools.SerializeMessage(cmd, credentials); err != nil; {
		}
		send_channel <- serialized
		rec_bytes := <-receive_channel
		var response tools.Message
		err = tools.Deserializejson(rec_bytes, &response)
		if err != nil {
			fmt.Println("[error] - error while le deserializing message: unknown response", err)
			continue
		}
		switch response.CMD {
		case "ok":
			if c == "1" {
				fmt.Println("Your Are Logged!")
				online(credentials, send_channel, receive_channel, error_channel)
			} else {
				fmt.Println("User registered successfully")
			}
		case "error":
			if c == "1" {
				fmt.Println("Unable to login:", response.DATA)
			} else {
				fmt.Println("Unable to register:", response.DATA)
			}
		default:
			fmt.Println("Unknown command...", response.CMD)
		}
	}
}

func initialPage() string {
	choice := ""
	for choice != "1" && choice != "2" && choice != "0" {
		fmt.Println("=-=-=-=-=-=-=-=-=-=-=-=-=--=-=--===-=-=-=-=-")
		fmt.Println("=-=-=-=-=-=-=-WizardDuel-=-=--===-=-=-=-=-=-")
		fmt.Println("=-=-=-=-=-=-=-=-=-=-=-=-=--=-=--===-=-=-=-=-")
		fmt.Println("1 - Login")
		fmt.Println("2 - SignIn")
		fmt.Println("0 - Exit")
		choice = tools.Input("> ")
		exec.Command("clear")
	}
	return choice
}

func online(credentials tools.UserCredentials, send_channel chan []byte, receive_channel chan []byte, error_channel chan error) {
	var choice = ""
	for choice != "1" && choice != "2" {
		fmt.Println("You are logged!")
		fmt.Println("1 - Find match")
		fmt.Println("0 - Exit")
    choice = tools.Input("> ")
    exec.Command("clear")
	}
  if choice == "0"{
    os.Exit(0)
  }
  sendRequest(tools.Play.String(), credentials, send_channel)
  for {
    select {
    case serialized := <-receive_channel:
      var receive tools.Message
      err := tools.Deserializejson(serialized, &receive)
      if (err != nil) {
        fmt.Println("[error] an error occourred", err)
        continue
      }
      switch receive.CMD {
      case "ok":
        fmt.Println("You are playing with: ", receive.DATA)
      case "error":
        fmt.Println("Unable to Play:", receive.DATA)
      default:
        fmt.Println("[error] unknown command:", receive.CMD)
      }
    case err := <-error_channel:
      fmt.Println("[error] an error occourred", err)
    }
  }

}

func sendRequest(cmd string, data any, send_channel chan []byte) {
	fmt.Println("[error] -", data)
	var response []byte
	var err error
	for response, err = tools.SerializeMessage(cmd, data); err != nil; {
	}
	send_channel <- response
}
