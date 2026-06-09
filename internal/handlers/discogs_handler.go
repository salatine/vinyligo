package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/salatine/vinyligo/internal/discogs"
	"github.com/salatine/vinyligo/internal/models"
)

var Formats = []string{
	"Lp Vinil",
	"Compacto Vinil",
	"CD",
	"DVD",
	"Fita K7 Cassete",
	"LD LaserDisc",
}

var artistasAnonimos = []string{"Various", "No Artist", "Unknown Artist"}

type DiscogsHandler struct {
	client       *discogs.Client
	inputHandler *InputHandler
}

func NewDiscogsHandler(token string, inputHandler *InputHandler) *DiscogsHandler {
	return &DiscogsHandler{
		client:       discogs.NewClient(token),
		inputHandler: inputHandler,
	}
}

func (h *DiscogsHandler) GetProductSuggestion() (*models.ProductSuggestion, error) {
	for {
		query := h.inputHandler.GetVinylCode()
		releases, err := h.client.Search(query)
		if err != nil {
			return nil, err
		}

		for {
			n := len(releases)
			if n > 5 {
				n = 5
			}

			defaultChoice := "0"
			if n == 0 {
				defaultChoice = "r"
				fmt.Println("nenhum resultado")
			}

			for i := 0; i < n; i++ {
				r := releases[i]
				catNo := r.Catno
				format := ""
				if len(r.Formats) > 0 {
					format = r.Formats[0].Name
				}
				year := r.Year
			if year == "" {
				year = "?"
			}
			fmt.Printf("%d: %s. Ano de lançamento: %s. Format: %s. País: %s. Código: %s\n", i, r.Title, year, format, r.Country, catNo)
			}

			fmt.Println("\nn: pular  r: pesquisar novamente  q: sair")
			choice := h.inputHandler.GetString(
				fmt.Sprintf("[%s]: ", defaultChoice),
				defaultChoice,
			)

			switch choice {
			case "n":
				return models.NewNullSuggestion(), nil
			case "q":
				return nil, nil
			case "r":
			default:
				index, err := strconv.Atoi(choice)
				if err != nil || index < 0 || index >= len(releases) {
					fmt.Println("opção inválida")
					continue
				}
				return h.buildSuggestion(releases[index])
			}
			break
		}
	}
}

func (h *DiscogsHandler) buildSuggestion(release discogs.Release) (*models.ProductSuggestion, error) {
	details, err := h.client.GetRelease(release.ID)
	if err != nil {
		return nil, err
	}

	var suggestionArtist string
	suggestionIsNational := false

	if len(details.Artists) > 0 {
		artist := details.Artists[0]
		isAnonymous := false
		for _, a := range artistasAnonimos {
			if artist.Name == a {
				isAnonymous = true
				break
			}
		}
		if !isAnonymous {
			suggestionArtist = artist.Name
			artistDetails, err := h.client.GetArtist(artist.ID)
			if err == nil {
				suggestionIsNational = strings.Contains(strings.ToLower(artistDetails.Profile), "brazil")
			}
		}
	}

	format := Formats[0]
	if len(details.Formats) > 0 {
		for _, f := range Formats {
			if strings.Contains(f, details.Formats[0].Name) {
				format = f
				break
			}
		}
	}

	lpsQuantity := 1
	if len(details.Formats) > 0 {
		if qty, err := strconv.Atoi(details.Formats[0].Qty); err == nil {
			lpsQuantity = qty
		}
	}

	var label string
	if len(details.Labels) > 0 {
		label = details.Labels[0].Name
	}

	album := details.Title
	if album == suggestionArtist {
		album = ""
	}

	isNew := false
	isRepeated := false
	isDoubleCovered := false
	songQty := len(details.Tracklist)
	albumDuration := details.GetAlbumDuration()
	isImported := details.Country != "Brazil"

	var releaseYear *int
	if details.Year != 0 {
		releaseYear = &details.Year
	}

	return &models.ProductSuggestion{
		Format:          &format,
		Artist:          &suggestionArtist,
		Album:           &album,
		LPsQuantity:     &lpsQuantity,
		Genres:          details.Genres,
		IsNew:           &isNew,
		IsNational:      &suggestionIsNational,
		IsRepeated:      &isRepeated,
		IsDoubleCovered: &isDoubleCovered,
		SongQuantity:    &songQty,
		AlbumDuration:   &albumDuration,
		ReleaseYear:     releaseYear,
		Label:           &label,
		Observation:     nil,
		IsImported:      &isImported,
		Stock:           nil,
	}, nil
}
