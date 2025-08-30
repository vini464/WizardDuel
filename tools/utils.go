package tools

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)


func Input(arg string) string {
  scanner := bufio.NewScanner(os.Stdin)
  fmt.Print(arg)
  scanner.Scan()
  return strings.TrimSpace(scanner.Text())
}
