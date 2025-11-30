package notify

import (
	"fmt"
	"os/exec"
)

func Send(appName string) {
	msg := fmt.Sprintf("You use %s to long, take a rest!", appName)
	exec.Command("notify-send", "AppMon", msg).Run()
}
