package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

// config for Iceburg
type config struct {
	ClientName         string            `yaml:"client_name"`
	DbConnectionString string            `yaml:"db_connection_string"`
	SessionFilename    string            `yaml:"session_filename"`
	Days               map[string]int    `yaml:"days"`
	MessageTemplates   map[string]string `yaml:"message_templates"`
}

func (c config) getMessageTemplate(key string) string {
	template, ok := c.MessageTemplates[key]
	if !ok {
		return ""
	}
	return template
}

func loadConfig(c *config) {
	f, err := os.Open("config.yml")
	checkError(err)
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&c)
	checkError(err)
}
