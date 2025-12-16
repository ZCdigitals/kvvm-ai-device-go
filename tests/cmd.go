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
		log.Println("cmd error", err)
		return
	}

	log.Println("cmd output", string(output))
}
