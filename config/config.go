package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type DiscogsConfig struct {
	Token string `json:"token"`
}

type GmailConfig struct {
	AppPassword string `json:"app_password"`
}

type ShopifyConfig struct {
	ShopName      string `json:"shop_name"`
	AccessToken   string `json:"access_token"`
	LocationID    string `json:"location_id"`
	PublicationID string `json:"publication_id"`
}

type MercadoLivreConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type ImgbbConfig struct {
	ApiKey string `json:"api_key"`
}

type Config struct {
	PicturesPath           string `json:"pictures_path"`
	ResumeDirectoryPath    string `json:"resume_directory_path"`
	JsonDirectoryPath      string `json:"json_directory_path"`
	Message                string `json:"message"`
	Sender                 string `json:"sender"`
	Receivers              []string `json:"receivers"`
	GoogleCredentialsPath  string `json:"google_credentials_path"`

	Discogs      DiscogsConfig      `json:"discogs"`
	Gmail        GmailConfig        `json:"gmail"`
	Shopify      ShopifyConfig      `json:"shopify"`
	MercadoLivre MercadoLivreConfig `json:"mercadolivre"`
	Imgbb        ImgbbConfig        `json:"imgbb"`
}

var AppConfig *Config

func Load(configPath string) error {
	file, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	AppConfig = &Config{}

	if err := json.NewDecoder(file).Decode(AppConfig); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	return nil
}

func GetResumeSheetPath() string {
	date := time.Now().Format("02.01.2006")
	os.MkdirAll(AppConfig.ResumeDirectoryPath, 0755)
	return filepath.Join(AppConfig.ResumeDirectoryPath, fmt.Sprintf("Relação %s.xlsx", date))
}

func GetJsonPath() string {
	date := time.Now().Format("02.01.2006")
	os.MkdirAll(AppConfig.JsonDirectoryPath, 0755)
	return filepath.Join(AppConfig.JsonDirectoryPath, fmt.Sprintf("produtos_%s.json", date))
}
