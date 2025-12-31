package wifi

import (
	"log"
	"os/exec"
	"strings"
)

type WifiStatus struct {
	Enable    bool
	Connected bool
}

func ReadWifiStatus() WifiStatus {
	status := WifiStatus{}

	// check if enable
	cmd := exec.Command(
		"nmcli",
		"-g",
		"wifi", "radio",
	)
	output, err := cmd.Output()
	if err != nil {
		log.Println("wifi read status wifi radio error", err)
		return status
	}
	status.Enable = strings.TrimSpace(string(output)) == "enabled"

	// check if connected
	cmd = exec.Command(
		"nmcli",
		"-g",
		"DEVICE,TYPE,STATE,CONNECTION",
		"device", "status",
	)
	output, err = cmd.Output()
	if err != nil {
		log.Println("wifi read status device status error", err)
		return status
	}
	lines := strings.SplitSeq(strings.TrimSpace(string(output)), "\n")
	for line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) != 4 {
			continue
		} else if strings.Contains(fields[2], "connected") {
			status.Connected = true
			break
		}
	}

	return status
}

func ConnectWifi(ssid string, password string) error {
	cmd := exec.Command(
		"nmcli",
		"device",
		"wifi",
		"connect", ssid,
		"password", password,
	)
	return cmd.Run()
}
