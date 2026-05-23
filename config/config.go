package config

import (
	log "HLTV-Manager/logger"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	SiteIP         string
	SitePort       string
	HltvDocker     string
	HltvRunnerFile string
	HltvDemosDir   string
	ApiToken       string
}

var data Config

func InitConfig() {
	godotenv.Load("config.env")

	data = Config{
		SiteIP:         os.Getenv("SITE_IP"),
		SitePort:       os.Getenv("SITE_PORT"),
		HltvDocker:     os.Getenv("HLTV_DOCKER"),
		HltvRunnerFile: os.Getenv("HLTV_RUNNER_FILE"),
		HltvDemosDir:   os.Getenv("HLT_DEMOS_DIR"),
		ApiToken:       os.Getenv("API_TOKEN"),
	}

	// TODO: Debug

	log.InfoLogger.Printf("The env configuration is loaded: %v", data)
}

func SiteIP() string {
	return data.SiteIP
}

func SitePort() string {
	return data.SitePort
}

func HltvDocker() string {
	return data.HltvDocker
}

func HltvRunnerFile() string {
	return data.HltvRunnerFile
}

func HltvDemosDir() string {
	return data.HltvDemosDir
}

func ApiToken() string {
    return data.ApiToken
}