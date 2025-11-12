package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)

	cmd := exec.Command(filepath.Join(dir, "cmd.sh"))

	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("cmd error %v", err)
		return
	}

	log.Printf("cmd output %s", string(output))
}
