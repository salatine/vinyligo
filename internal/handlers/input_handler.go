package handlers

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/salatine/vinyligo/internal/models"
)

type InputHandler struct {
	scanner *bufio.Scanner
}

func NewInputHandler() *InputHandler {
	return &InputHandler{scanner: bufio.NewScanner(os.Stdin)}
}

func (h *InputHandler) GetString(prompt string, defaultValue ...string) string {
	if len(defaultValue) > 0 && defaultValue[0] != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue[0])
	} else {
		fmt.Printf("%s: ", prompt)
	}
	h.scanner.Scan()
	text := h.scanner.Text()
	if text == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return text
}

func (h *InputHandler) GetVinylCode() string {
	return h.GetString("pesquisa do vinil")
}

func (h *InputHandler) GetInt(prompt string, defaultValue ...int) int {
	for {
		var input string
		if len(defaultValue) > 0 {
			input = h.GetString(prompt, strconv.Itoa(defaultValue[0]))
		} else {
			input = h.GetString(prompt)
		}
		val, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("valor inválido, tente novamente!")
			continue
		}
		return val
	}
}

func (h *InputHandler) GetFloat(prompt string, defaultValue ...float64) float64 {
	for {
		var input string
		if len(defaultValue) > 0 {
			input = h.GetString(prompt, fmt.Sprintf("%.0f", defaultValue[0]))
		} else {
			input = h.GetString(prompt)
		}
		val, err := strconv.ParseFloat(input, 64)
		if err != nil {
			fmt.Println("valor inválido, tente novamente!")
			continue
		}
		return val
	}
}

func (h *InputHandler) GetBool(prompt string, defaultValue ...bool) bool {
	var defStr string
	if len(defaultValue) > 0 {
		if defaultValue[0] {
			defStr = "s"
		} else {
			defStr = "n"
		}
	}

	for {
		var input string
		if defStr != "" {
			input = h.GetString(prompt, defStr)
		} else {
			input = h.GetString(prompt)
		}
		input = strings.ToLower(input)
		switch input {
		case "s", "sim":
			return true
		case "n", "não", "nao":
			return false
		default:
			fmt.Println("valor inválido, tente novamente!")
		}
	}
}

func (h *InputHandler) Confirm(prompt string) bool {
	input := strings.ToLower(h.GetString(prompt, "s"))
	return input != "n"
}

func (h *InputHandler) DisplayProductInformation(product *models.Product) {
	fmt.Printf("\tformato: %s\n", product.Format)
	fmt.Printf("\tnome do artista: %s\n", product.Artist)
	fmt.Printf("\tnome do álbum: %s\n", product.Album)
	fmt.Printf("\tpreço: R$%.2f\n", product.Price)
	fmt.Printf("\tquantidade de encartes: %d\n", product.GatefoldQuantity)
	fmt.Printf("\tquantidade de discos: %d\n", product.LPsQuantity)
	fmt.Printf("\tgênero(s): %v\n", product.Genres)
	fmt.Printf("\tnovo: %t\n", product.IsNew)
	fmt.Printf("\tnacional: %t\n", product.IsNational)
	fmt.Printf("\trepetido: %t\n", product.IsRepeated)
	fmt.Printf("\testoque: %d\n", product.Stock)
	fmt.Printf("\tcapa dupla: %t\n", product.IsDoubleCovered)
	if product.Observation != nil && *product.Observation != "" {
		fmt.Printf("\tobservação: %s\n", *product.Observation)
	}
	if product.ReleaseYear != nil {
		fmt.Printf("\tano: %d\n", *product.ReleaseYear)
	}
	if product.Label != nil && *product.Label != "" {
		fmt.Printf("\tgravadora: %s\n", *product.Label)
	}
	if product.IsImported != nil {
		fmt.Printf("\timportado: %t\n", *product.IsImported)
	}
	if product.SongQuantity != nil {
		fmt.Printf("\tmúsicas: %d\n", *product.SongQuantity)
	}
	if product.AlbumDuration != nil && *product.AlbumDuration > 0 {
		fmt.Printf("\tduração: %.1f min\n", *product.AlbumDuration)
	}
	if len(product.Pictures) > 0 {
		fmt.Printf("\tfotos: %d\n", len(product.Pictures))
	}
	if product.PublishTo != "" {
		fmt.Printf("\tplataforma: %s\n", product.PublishTo)
	}
}

func (h *InputHandler) CheckProducts(products []*models.Product) {
	for _, product := range products {
		fmt.Println(product.Title(func(s string) string { return s }))
	}
	fmt.Println()
}
