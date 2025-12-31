package src

import (
	"encoding/json"
	"os"
	"time"
)

type Config struct {
	ID string `json:"id"`

	AccessToken           string    `json:"accessToken"`
	AccessTokenExpiresAt  time.Time `json:"accessTokenExpiresAt"`
	RefreshToken          string    `json:"refreshToken"`
	RefreshTokenExpiresAt time.Time `json:"refreshTokenExpiresAt"`

	WakeOnLanMac string `json:"wakeOnLanMac"`
}

type ConfigFile struct {
	path string

	Config Config
}

func (cf *ConfigFile) Load() error {
	_, err := os.Stat(cf.path)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(cf.path)
	if err != nil {
		return err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return err
	}

	cf.Config = config

	return nil
}

func (cf *ConfigFile) Save() error {
	data, err := json.MarshalIndent(cf.Config, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(cf.path, data, 0644)
	if err != nil {
		return err
	}

	return nil
}
