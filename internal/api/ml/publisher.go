package ml

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/salatine/vinyligo/internal/models"
)

var MeasuresByFormat = map[string]Measures{
	"Lp Vinil":        {40.0, 5.0, 40.0, 500.0},
	"Compacto Vinil":  {22.0, 4.0, 22.0, 200.0},
	"CD":              {16.0, 2.0, 14.0, 100.0},
	"DVD":             {19.0, 1.5, 13.5, 100.0},
	"Fita K7 Cassete": {11.0, 2.0, 7.0, 60.0},
	"LD LaserDisc":    {32.0, 5.0, 32.0, 600.0},
}

type Measures struct {
	Width  float64
	Height float64
	Depth  float64
	Weight float64
}

type MLPicture struct {
	Source string `json:"source"`
}

type MLShipping struct {
	Mode        string `json:"mode"`
	LocalPickUp bool   `json:"local_pick_up"`
}

type MLAttribute struct {
	ID        string `json:"id"`
	ValueName string `json:"value_name"`
}

type MLDescription struct {
	PlainText string `json:"plain_text"`
}

type MLListing struct {
	Title        string        `json:"title"`
	CategoryID   string        `json:"category_id"`
	Price        float64       `json:"price"`
	CurrencyID   string        `json:"currency_id"`
	AvailableQty int           `json:"available_quantity"`
	BuyingMode   string        `json:"buying_mode"`
	Condition    string        `json:"condition"`
	ListingType  string        `json:"listing_type_id"`
	Pictures     []MLPicture   `json:"pictures"`
	Shipping     MLShipping    `json:"shipping"`
	Attributes   []MLAttribute `json:"attributes"`
}

type MLListingResponse struct {
	ID        string `json:"id"`
	Permalink string `json:"permalink"`
}

func (c *Client) PublishProduct(
	product *models.Product,
	titleEditor func(string) string,
	pictureUploader func(string) (string, error),
) (*MLListingResponse, error) {
	listing, err := c.buildListing(product, titleEditor, pictureUploader)
	if err != nil {
		return nil, err
	}

	respBody, err := c.Post("/items", listing)
	if err != nil {
		return nil, err
	}

	var mlResponse MLListingResponse
	if err := json.Unmarshal(respBody, &mlResponse); err != nil {
		return nil, err
	}

	desc := MLDescription{PlainText: product.Description()}
	_, err = c.Post(fmt.Sprintf("/items/%s/description", mlResponse.ID), desc)
	if err != nil {
		return nil, err
	}

	return &mlResponse, nil
}

func (c *Client) buildListing(
	product *models.Product,
	titleEditor func(string) string,
	pictureUploader func(string) (string, error),
) (*MLListing, error) {
	pictureURLs, err := product.GetPictureURLs(pictureUploader)
	if err != nil {
		return nil, err
	}

	pictures := make([]MLPicture, 0, len(pictureURLs))
	for _, url := range pictureURLs {
		pictures = append(pictures, MLPicture{Source: url})
	}

	condition := "used"
	if product.IsNew {
		condition = "new"
	}

	label := "N/A"
	if product.Label != nil {
		label = *product.Label
	}

	artist := product.Artist
	if artist == "" {
		artist = product.Album
	}

	album := product.Album
	if strings.TrimSpace(album) == "" {
		album = product.Artist
	}

	attributes := []MLAttribute{
		{ID: "ARTIST_NAME", ValueName: artist},
		{ID: "MUSIC_ARTIST_NAME", ValueName: artist},
		{ID: "ALBUM_NAME", ValueName: album},
		{ID: "FAMILY_NAME", ValueName: artist},
		{ID: "PRODUCTION_COMPANY", ValueName: label},
		{ID: "FORMAT", ValueName: "Físico"},
		{ID: "ALBUM_TYPE", ValueName: albumType(product)},
		{ID: "INCLUDES_ADDITIONAL_TRACKS", ValueName: "Não"},
		{ID: "CONDITION", ValueName: "Usado"},
	}

	if product.ReleaseYear != nil {
		attributes = append(attributes, MLAttribute{
			ID:        "RELEASE_YEAR",
			ValueName: strconv.Itoa(*product.ReleaseYear),
		})
	}

	if product.SongQuantity != nil {
		attributes = append(attributes, MLAttribute{
			ID:        "TRACKS_QUANTITY",
			ValueName: fmt.Sprintf("%d", *product.SongQuantity),
		})
	}

	return &MLListing{
		Title:        product.Title(titleEditor),
		CategoryID:   "MLB1174",
		Price:        product.Price,
		CurrencyID:   "BRL",
		AvailableQty: product.Stock,
		BuyingMode:   "buy_it_now",
		Condition:    condition,
		ListingType:  "gold_special",
		Pictures:     pictures,
		Shipping:     MLShipping{Mode: "me2", LocalPickUp: false},
		Attributes:   attributes,
	}, nil
}

func albumType(product *models.Product) string {
	if product.LPsQuantity > 1 {
		return fmt.Sprintf("%d %ss", product.LPsQuantity, strings.ReplaceAll(product.Format, " Vinil", ""))
	}
	return product.Format
}
