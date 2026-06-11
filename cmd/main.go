package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/salatine/vinyligo/config"
	"github.com/salatine/vinyligo/internal/api/ml"
	"github.com/salatine/vinyligo/internal/api/shopify"
	"github.com/salatine/vinyligo/internal/email"
	"github.com/salatine/vinyligo/internal/handlers"
	"github.com/salatine/vinyligo/internal/models"
	"github.com/salatine/vinyligo/internal/sheets"
	"github.com/salatine/vinyligo/pkg/utils"
)

const (
	ProductsBackupFile = "./products.json"
	ConfigFile         = "./config.json"
)

var Formats = []string{
	"Lp Vinil",
	"Compacto Vinil",
	"CD",
	"DVD",
	"Fita K7 Cassete",
	"LD LaserDisc",
}

func main() {
	if err := config.Load(ConfigFile); err != nil {
		log.Fatalf("erro: %v", err)
	}

	input := handlers.NewInputHandler()

	pictureHandler := handlers.NewPictureHandler(config.AppConfig.Imgbb.ApiKey)

	discogsHandler := handlers.NewDiscogsHandler(config.AppConfig.Discogs.Token, input)

	products := getJSONProducts(input)
	products = createProducts(input, discogsHandler, pictureHandler, products)

	products = confirmProducts(input, products)

	fmt.Println("\nverificando títulos...")
	for _, p := range products {
		title := p.Title(utils.EditTitle)
		p.TitleOverride = &title
		desc := p.Description()
		p.DescriptionOverride = &desc
	}

	saveJSONProducts(products)

	if config.AppConfig.JsonDirectoryPath != "" {
		data, _ := json.MarshalIndent(products, "", "  ")
		jsonPath := config.GetJsonPath()
		if err := os.WriteFile(jsonPath, data, 0644); err != nil {
			log.Printf("erro salvando json: %v", err)
		} else {
			fmt.Printf("json salvo em %s\n", jsonPath)
		}
	}

	needsML := false
	needsShopify := false
	for _, p := range products {
		if p.PublishTo == "ambos" || p.PublishTo == "ml" {
			needsML = true
		}
		if p.PublishTo == "ambos" || p.PublishTo == "shopify" {
			needsShopify = true
		}
	}

	var wg sync.WaitGroup

	if needsML {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("\npublicando no mercado livre...")
			mlClient := ml.NewClient(config.AppConfig.MercadoLivre.ClientID, config.AppConfig.MercadoLivre.ClientSecret)
			for i, product := range products {
				if product.PublishTo == "shopify" || product.PublishTo == "nenhum" {
					continue
				}
				fmt.Printf("ml %d/%d: %s\n", i+1, len(products), product.Title(utils.EditTitle))
				resp, err := mlClient.PublishProduct(product, utils.EditTitle, pictureHandler.UploadPicture)
				if err != nil {
					log.Printf("  ml erro: %v", err)
					continue
				}
				fmt.Printf("  ml publicado: %s\n", resp.ID)
			}
		}()
	}

	if needsShopify {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("\npublicando no shopify...")
			shopifyClient := shopify.NewClient(
				config.AppConfig.Shopify.ShopName,
				config.AppConfig.Shopify.AccessToken,
				config.AppConfig.Shopify.LocationID,
				config.AppConfig.Shopify.PublicationID,
			)
			for i, product := range products {
				if product.PublishTo == "ml" || product.PublishTo == "nenhum" {
					continue
				}
				fmt.Printf("shopify %d/%d: %s\n", i+1, len(products), product.Title(utils.EditTitle))
				id, err := shopifyClient.PublishProduct(product, utils.EditTitle, pictureHandler.UploadPicture)
				if err != nil {
					log.Printf("  shopify erro: %v", err)
					continue
				}
				fmt.Printf("  shopify publicado: %s\n", id)
			}
		}()
	}

	wg.Wait()

	fmt.Println("\ncriando relação...")
	resumeSheet := sheets.NewResumeSheet()
	if err := resumeSheet.CreateResumeSheet(products, config.GetResumeSheetPath(), utils.EditTitle); err != nil {
		log.Fatalf("erro criando relação: %v", err)
	}
	fmt.Printf("relação salva em %s\n", config.GetResumeSheetPath())

	sendEmailPrompt(input, products)
}

func createProducts(input *handlers.InputHandler, discogsHandler *handlers.DiscogsHandler, pictureHandler *handlers.PictureHandler, products []*models.Product) []*models.Product {
	if len(products) > 0 && !input.Confirm("deseja cadastrar mais produtos? [S/n]") {
		return products
	}

	for {
		suggestion, err := discogsHandler.GetProductSuggestion()
		if err != nil {
			log.Printf("erro discogs: %v", err)
			continue
		}
		if suggestion == nil {
			break
		}

		product := createProduct(input, suggestion, pictureHandler)
		product.PublishTo = "ambos"

		fmt.Println("\nproduto cadastrado:")
		input.DisplayProductInformation(product)

		if input.GetBool("deseja alterar algum valor? (s/N)", false) {
			product = changeProductValues(input, product, suggestion)
		}

		products = append(products, product)
		saveJSONProducts(products)

		if !input.Confirm("deseja cadastrar mais produtos? [S/n]") {
			break
		}
	}

	return products
}

func confirmProducts(input *handlers.InputHandler, products []*models.Product) []*models.Product {
	edited := editProductsInEditor(products)
	if edited != nil {
		products = edited
	}

	for i, p := range products {
		fmt.Printf("  %d. %s - %s | R$%.2f | %s\n", i+1, p.Artist, p.Album, p.Price, p.PublishTo)
	}
	fmt.Println()

	return products
}

func editProductsInEditor(products []*models.Product) []*models.Product {
	type editableProduct struct {
		Titulo     string   `json:"titulo"`
		Descricao  string   `json:"descricao"`
		Formato    string   `json:"formato"`
		Artista    string   `json:"artista"`
		Album      string   `json:"album"`
		Preco      float64  `json:"preco"`
		Encartes   int      `json:"encartes"`
		Discos     int      `json:"discos"`
		Generos    []string `json:"generos"`
		Novo       bool     `json:"novo"`
		Nacional   bool     `json:"nacional"`
		Repetido   bool     `json:"repetido"`
		Estoque    int      `json:"estoque"`
		CapaDupla  bool     `json:"capa_dupla"`
		Observacao string   `json:"observacao,omitempty"`
		Importado  bool     `json:"importado"`
		Plataforma string   `json:"plataforma"`
	}

	identity := func(s string) string { return s }

	var editable []editableProduct
	for _, p := range products {
		ep := editableProduct{
			Titulo:     p.Title(identity),
			Descricao:  p.Description(),
			Formato:    p.Format,
			Artista:    p.Artist,
			Album:      p.Album,
			Preco:      p.Price,
			Encartes:   p.GatefoldQuantity,
			Discos:     p.LPsQuantity,
			Generos:    p.Genres,
			Novo:       p.IsNew,
			Nacional:   p.IsNational,
			Repetido:   p.IsRepeated,
			Estoque:    p.Stock,
			CapaDupla:  p.IsDoubleCovered,
			Plataforma: p.PublishTo,
		}
		if p.Observation != nil {
			ep.Observacao = *p.Observation
		}
		if p.IsImported != nil {
			ep.Importado = *p.IsImported
		}
		editable = append(editable, ep)
	}

	data, _ := json.MarshalIndent(editable, "", "  ")

	tmpFile, err := os.CreateTemp("", "vinyligo-*.json")
	if err != nil {
		log.Printf("erro criando arquivo temporário: %v", err)
		return nil
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	tmpFile.Write(data)
	tmpFile.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("erro abrindo editor: %v", err)
		return nil
	}

	newData, err := os.ReadFile(tmpPath)
	if err != nil {
		log.Printf("erro lendo arquivo editado: %v", err)
		return nil
	}

	var edited []editableProduct
	if err := json.Unmarshal(newData, &edited); err != nil {
		log.Printf("erro parseando json editado: %v", err)
		return nil
	}

	for i, ep := range edited {
		if i >= len(products) {
			break
		}
		products[i].Format = ep.Formato
		products[i].Artist = ep.Artista
		products[i].Album = ep.Album
		products[i].Price = ep.Preco
		products[i].GatefoldQuantity = ep.Encartes
		products[i].LPsQuantity = ep.Discos
		products[i].Genres = ep.Generos
		products[i].IsNew = ep.Novo
		products[i].IsNational = ep.Nacional
		products[i].IsRepeated = ep.Repetido
		products[i].Stock = ep.Estoque
		products[i].IsDoubleCovered = ep.CapaDupla
		products[i].PublishTo = ep.Plataforma
		if ep.Observacao != "" {
			products[i].Observation = &ep.Observacao
		} else {
			products[i].Observation = nil
		}
		imported := ep.Importado
		products[i].IsImported = &imported

		computedTitle := products[i].Title(identity)
		if ep.Titulo != computedTitle {
			override := ep.Titulo
			products[i].TitleOverride = &override
		}
		computedDesc := products[i].Description()
		if ep.Descricao != computedDesc {
			override := ep.Descricao
			products[i].DescriptionOverride = &override
		}
	}

	return products
}

func createProduct(input *handlers.InputHandler, suggestion *models.ProductSuggestion, pictureHandler *handlers.PictureHandler) *models.Product {
	format := getFormat(input, suggestion)
	isNew := getIsNew(input, suggestion)

	artist := ""
	if suggestion.Artist != nil {
		artist = input.GetString("nome do artista", *suggestion.Artist)
	} else {
		artist = input.GetString("nome do artista")
	}

	album := ""
	if suggestion.Album != nil {
		album = input.GetString("nome do álbum", *suggestion.Album)
	} else {
		album = input.GetString("nome do álbum")
	}

	price := input.GetFloat("preço", 30)
	gatefoldQty := input.GetInt("quantidade de encartes", 0)

	lpsQty := 1
	if suggestion.LPsQuantity != nil {
		lpsQty = input.GetInt("quantidade de discos", *suggestion.LPsQuantity)
	} else {
		lpsQty = input.GetInt("quantidade de discos", 1)
	}

	genres := getGenres(input, suggestion)
	observation := input.GetString("observação", "")
	isNational := getIsNational(input, suggestion)
	isRepeated := getIsRepeated(input, suggestion)

	stock := 1
	if isNew {
		if suggestion.Stock != nil {
			stock = input.GetInt("unidades", *suggestion.Stock)
		} else {
			stock = input.GetInt("unidades", 1)
		}
	}

	isDoubleCovered := getIsDoubleCovered(input, suggestion)
	pictures := getPictures(input, format, isNew)

	songQty := 1
	if suggestion.SongQuantity != nil {
		songQty = *suggestion.SongQuantity
	}
	albumDuration := 0.0
	if suggestion.AlbumDuration != nil {
		albumDuration = *suggestion.AlbumDuration
	}
	var releaseYear *int
	if suggestion.ReleaseYear != nil {
		releaseYear = suggestion.ReleaseYear
	}
	var label *string
	if suggestion.Label != nil {
		label = suggestion.Label
	}
	var isImported *bool
	if suggestion.IsImported != nil {
		isImported = suggestion.IsImported
	}

	var obs *string
	if observation != "" {
		obs = &observation
	}

	return &models.Product{
		Format:           format,
		Artist:           artist,
		Album:            album,
		Price:            price,
		GatefoldQuantity: gatefoldQty,
		LPsQuantity:      lpsQty,
		Genres:           genres,
		IsNew:            isNew,
		IsNational:       isNational,
		IsRepeated:       isRepeated,
		Stock:            stock,
		IsDoubleCovered:  isDoubleCovered,
		Pictures:         pictures,
		SongQuantity:     &songQty,
		AlbumDuration:    &albumDuration,
		ReleaseYear:      releaseYear,
		Label:            label,
		Observation:      obs,
		IsImported:       isImported,
	}
}

func getFormat(input *handlers.InputHandler, suggestion *models.ProductSuggestion) string {
	fmt.Println()
	for i, f := range Formats {
		fmt.Printf("%d: %s\n", i, f)
	}

	defaultIdx := "0"
	if suggestion.Format != nil {
		for i, f := range Formats {
			if f == *suggestion.Format {
				defaultIdx = fmt.Sprintf("%d", i)
				break
			}
		}
	}

	for {
		choice := input.GetString("formato", defaultIdx)
		idx, err := fmt.Sscanf(choice, "%d", new(int))
		if err != nil || idx == 0 {
			fmt.Printf("valor inválido, selecione um número entre 0 e %d\n", len(Formats)-1)
			continue
		}
		var n int
		fmt.Sscanf(choice, "%d", &n)
		if n < 0 || n >= len(Formats) {
			fmt.Printf("valor inválido, selecione um número entre 0 e %d\n", len(Formats)-1)
			continue
		}
		return Formats[n]
	}
}

func getIsNew(input *handlers.InputHandler, suggestion *models.ProductSuggestion) bool {
	def := false
	if suggestion.IsNew != nil {
		def = *suggestion.IsNew
	}
	return input.GetBool("novo (s/n)", def)
}

func getIsNational(input *handlers.InputHandler, suggestion *models.ProductSuggestion) bool {
	def := false
	if suggestion.IsNational != nil {
		def = *suggestion.IsNational
	}
	return input.GetBool("nacional (s/n)", def)
}

func getIsRepeated(input *handlers.InputHandler, suggestion *models.ProductSuggestion) bool {
	def := false
	if suggestion.IsRepeated != nil {
		def = *suggestion.IsRepeated
	}
	return input.GetBool("repetido (s/n)", def)
}

func getIsDoubleCovered(input *handlers.InputHandler, suggestion *models.ProductSuggestion) bool {
	def := false
	if suggestion.IsDoubleCovered != nil {
		def = *suggestion.IsDoubleCovered
	}
	return input.GetBool("capa dupla (s/N)", def)
}

func getGenres(input *handlers.InputHandler, suggestion *models.ProductSuggestion) []string {
	if len(suggestion.Genres) > 0 {
		fmt.Println("selecione gêneros e/ou digite novos, separados por vírgula:")
		for i, g := range suggestion.Genres {
			fmt.Printf("\t%d: %s\n", i, g)
		}
	}

	defaultGenre := ""
	if len(suggestion.Genres) > 0 {
		defaultGenre = "0"
	}

	choices := input.GetString("gênero", defaultGenre)
	var genres []string
	for _, choice := range strings.Split(choices, ",") {
		choice = strings.TrimSpace(choice)
		if idx, err := fmt.Sscanf(choice, "%d", new(int)); err == nil && idx > 0 {
			var n int
			fmt.Sscanf(choice, "%d", &n)
			if n >= 0 && n < len(suggestion.Genres) {
				genres = append(genres, suggestion.Genres[n])
			}
		} else if choice != "" {
			genres = append(genres, choice)
		}
	}
	if len(genres) == 0 && len(suggestion.Genres) > 0 {
		genres = append(genres, suggestion.Genres[0])
	}
	return genres
}

func getPictures(input *handlers.InputHandler, format string, isNew bool) []string {
	var pictures []string

	if runtime.GOOS == "windows" {
		pictures = getPicturesWindows()
	} else {
		pictures = getPicturesLinux(input)
	}

	if format == "Lp Vinil" && !isNew && len(pictures) > 1 {
		pictures = append(pictures[1:], pictures[0])
	}

	return pictures
}

func getPicturesLinux(input *handlers.InputHandler) []string {
	raw := input.GetString("drag n' drop")
	if raw == "" {
		return nil
	}

	raw = strings.ReplaceAll(raw, "'", "")
	raw = strings.ReplaceAll(raw, "\"", "")
	raw = strings.ReplaceAll(raw, "C:\\", "/mnt/c/")
	raw = strings.ReplaceAll(raw, "\\", "/")
	raw = strings.TrimSpace(raw)

	return strings.Split(raw, " ")
}

func getPicturesWindows() []string {
	cmd := exec.Command("powershell", "-Command", `
Add-Type -AssemblyName System.Windows.Forms
$dialog = New-Object System.Windows.Forms.OpenFileDialog
$dialog.Filter = "Image files (*.png;*.jpg;*.jpeg;*.webp)|*.png;*.jpg;*.jpeg;*.webp"
$dialog.Multiselect = $true
$dialog.InitialDirectory = "`+config.AppConfig.PicturesPath+`"
if ($dialog.ShowDialog() -eq 'OK') { $dialog.FileNames -join "`+"\n"+`" }
`)
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return nil
	}
	var pictures []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			pictures = append(pictures, line)
		}
	}
	return pictures
}

func changeProductValues(input *handlers.InputHandler, product *models.Product, suggestion *models.ProductSuggestion) *models.Product {
	for {
		fmt.Println("\nqual valor deseja alterar?")
		fmt.Println("\tq: sair")
		fmt.Println("\t0: nome do artista")
		fmt.Println("\t1: nome do álbum")
		fmt.Println("\t2: preço")
		fmt.Println("\t3: quantidade de encartes")
		fmt.Println("\t4: quantidade de discos")
		fmt.Println("\t5: gênero(s)")
		fmt.Println("\t6: nacional")
		fmt.Println("\t7: repetido")
		fmt.Println("\t8: capa dupla")
		fmt.Println("\t9: formato")
		fmt.Println("\t10: observação")
		fmt.Printf("\t11: plataforma [%s]\n", product.PublishTo)

		choice := input.GetString("qual valor deseja alterar?", "0")

		switch choice {
		case "q":
			return product
		case "0":
			if suggestion.Artist != nil {
				product.Artist = input.GetString("nome do artista", *suggestion.Artist)
			} else {
				product.Artist = input.GetString("nome do artista", product.Artist)
			}
		case "1":
			if suggestion.Album != nil {
				product.Album = input.GetString("nome do álbum", *suggestion.Album)
			} else {
				product.Album = input.GetString("nome do álbum", product.Album)
			}
		case "2":
			product.Price = input.GetFloat("preço", product.Price)
		case "3":
			product.GatefoldQuantity = input.GetInt("quantidade de encartes", product.GatefoldQuantity)
		case "4":
			product.LPsQuantity = input.GetInt("quantidade de discos", product.LPsQuantity)
		case "5":
			product.Genres = getGenres(input, suggestion)
		case "6":
			product.IsNational = getIsNational(input, suggestion)
		case "7":
			product.IsRepeated = getIsRepeated(input, suggestion)
		case "8":
			product.IsDoubleCovered = getIsDoubleCovered(input, suggestion)
		case "9":
			product.Format = getFormat(input, suggestion)
		case "10":
			obs := input.GetString("observação", "")
			if obs != "" {
				product.Observation = &obs
			} else {
				product.Observation = nil
			}
		case "11":
			fmt.Println("0: ambos  1: ml  2: shopify  3: nenhum")
			p := input.GetString("plataforma", "0")
			switch p {
			case "1":
				product.PublishTo = "ml"
			case "2":
				product.PublishTo = "shopify"
			case "3":
				product.PublishTo = "nenhum"
			default:
				product.PublishTo = "ambos"
			}
		default:
			fmt.Println("valor inválido, tente novamente!")
			continue
		}

		fmt.Println("\nproduto atualizado:")
		input.DisplayProductInformation(product)

		if !input.Confirm("deseja alterar mais algum valor? [S/n]") {
			break
		}
	}
	return product
}

func getJSONProducts(input *handlers.InputHandler) []*models.Product {
	if _, err := os.Stat(ProductsBackupFile); os.IsNotExist(err) {
		return nil
	}
	if !input.Confirm("foi detectado um backup dos produtos, deseja carregá-lo? [S/n]") {
		return nil
	}
	data, err := os.ReadFile(ProductsBackupFile)
	if err != nil {
		log.Printf("erro lendo backup: %v", err)
		return nil
	}
	var products []*models.Product
	if err := json.Unmarshal(data, &products); err != nil {
		log.Printf("erro parsing backup: %v", err)
		return nil
	}
	for _, p := range products {
		if p.PublishTo == "" {
			p.PublishTo = "ambos"
		}
	}
	fmt.Printf("%d produtos carregados do arquivo de backup\n", len(products))
	return products
}

func saveJSONProducts(products []*models.Product) {
	data, err := json.MarshalIndent(products, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(ProductsBackupFile, data, 0644)
}

func sendEmailPrompt(input *handlers.InputHandler, products []*models.Product) {
	receivers := strings.Join(config.AppConfig.Receivers, ", ")
	if !input.Confirm(fmt.Sprintf("deseja enviar um e-mail com a relação para %s? [S/n]", receivers)) {
		return
	}

	emailConfig := email.EmailConfig{
		Sender:      config.AppConfig.Sender,
		Receivers:   config.AppConfig.Receivers,
		AppPassword: config.AppConfig.Gmail.AppPassword,
	}

	data := time.Now().Format("02/01/2006")
	if err := email.SendEmail(
		fmt.Sprintf("Relação %s", data),
		config.AppConfig.Message,
		emailConfig,
		config.GetResumeSheetPath(),
	); err != nil {
		log.Printf("erro enviando email: %v", err)
		return
	}
	fmt.Println("e-mail enviado com sucesso!")
}
