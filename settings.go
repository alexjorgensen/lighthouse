package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// settingsFile contains the configuration filename
const filename = "lighthouse.toml"

type Settings struct {
	SaveRequestTokenToDisk bool `toml:"SaveRequestTokenToDisk"`
	Database               struct {
		Name     string `toml:"Name"`
		HostName string `toml:"HostName"`
		Username string `toml:"Username"`
		Password string `toml:"Password"`
	} `toml:"Database"`
	NorlysAPI struct {
		URL                  string `toml:"URL"`
		UpdatePricesInterval int    `toml:"UpdatePricesInterval"`
	} `toml:"NorlysAPI"`
	ElOverblik struct {
		FetchDataFromElOverblik bool   `toml:"FetchDataFromElOverblik"`
		LighthouseToken         string `toml:"LighthouseToken"`
	} `toml:"ElOverblik"`
}

// ReadConfigurationFile find the configuration file on disk an parses it
func (s *Settings) ReadConfigurationFile() error {

	// find the directory of the application
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}
	if !fileExists(filepath.Join(dir, filename)) {
		return errors.New("error reading configuration file, file: " + filename + " does not exists")
	}

	// Read the configuration file into mem
	file, err := os.Open(filepath.Join(dir, filename))
	if err != nil {
		return errors.New("error opening configuration file, file: " + err.Error())
	}
	defer file.Close()
	tomlData, err := ioutil.ReadAll(file)
	if err != nil {
		return errors.New("ERROR reading configuration file, file: " + err.Error())
	}

	if _, err := toml.Decode(string(tomlData), &s); err != nil {
		return errors.New("ERROR decoding toml data:" + err.Error())
	}

	// Check if all the critical fields are configured correctly
	if s.Database.Name == "" {
		return errors.New("database name not configured")
	}
	if s.Database.Password == "" {
		return errors.New("database password not configured")
	}
	if s.Database.Username == "" {
		return errors.New("database username not configured")
	}
	if s.Database.HostName == "" {
		return errors.New("database hostname not configured")
	}
	if s.NorlysAPI.URL == "" {
		return errors.New("norlys url not configured")
	}

	if s.NorlysAPI.UpdatePricesInterval == 0 {
		s.NorlysAPI.UpdatePricesInterval = 3600
	}

	return nil
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
