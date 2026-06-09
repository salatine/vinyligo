package models

import (
	"fmt"
	"strconv"
	"strings"
)

const MLCharacterLimit = 60

var QuantityTranslation = map[int]string{
	2: "Duplo", 3: "Triplo", 4: "Quádruplo", 5: "Quíntuplo",
	6: "Sêxtuplo", 7: "Sétuplo", 8: "Óctuplo", 9: "Nônuplo",
}

type Product struct {
	Format              string
	Artist              string
	Album               string
	Price               float64
	GatefoldQuantity    int
	LPsQuantity         int
	Genres              []string
	IsNew               bool
	IsNational          bool
	IsRepeated          bool
	Stock               int
	IsDoubleCovered     bool
	Pictures            []string
	SongQuantity        *int
	AlbumDuration       *float64
	ReleaseYear         *int
	Label               *string
	Observation         *string
	IsImported          *bool
	PublishTo           string
	TitleOverride       *string
	DescriptionOverride *string

	pictureURLs []string
}

func (p *Product) GetPictureURLs(uploader func(string) (string, error)) ([]string, error) {
	if p.pictureURLs != nil {
		return p.pictureURLs, nil
	}

	type result struct {
		index int
		url   string
		err   error
	}

	ch := make(chan result, len(p.Pictures))
	for i, picture := range p.Pictures {
		go func(idx int, path string) {
			url, err := uploader(path)
			ch <- result{idx, url, err}
		}(i, picture)
	}

	urls := make([]string, len(p.Pictures))
	for range p.Pictures {
		r := <-ch
		if r.err != nil {
			return nil, fmt.Errorf("foto %d: %w", r.index+1, r.err)
		}
		urls[r.index] = r.url
	}

	p.pictureURLs = urls
	return urls, nil
}

func (p *Product) Description() string {
	if p.DescriptionOverride != nil {
		return *p.DescriptionOverride
	}
	var desc string
	if p.IsNew {
		desc = "PRODUTO NOVO, LACRADO."
	} else {
		desc = "PRODUTO USADO EM BOM ESTADO."
	}

	gatefold := ""
	if p.GatefoldQuantity >= 2 {
		gatefold = "COM ENCARTES. "
	} else if p.GatefoldQuantity == 1 {
		gatefold = "COM ENCARTE. "
	}

	lps := ""
	if p.LPsQuantity > 1 && p.LPsQuantity < 10 {
		lps = fmt.Sprintf("DISCO %s. ", strings.ToUpper(QuantityTranslation[p.LPsQuantity]))
	}
	if p.IsDoubleCovered {
		lps += "CAPA DUPLA."
	}
	if p.IsImported != nil && *p.IsImported {
		lps += " IMPORTADO."
	}

	obs := ""
	if p.Observation != nil {
		obs = *p.Observation
	}

	return fmt.Sprintf("%s\n%s %s\n%s", desc, gatefold, lps, obs)
}

func (p *Product) NationalityText() string {
	if p.IsNational {
		return "Brasil"
	}
	return "Internacional"
}

func (p *Product) Title(editFunc func(string) string) string {
	if p.TitleOverride != nil {
		return *p.TitleOverride
	}
	album := p.Album
	if album == p.Artist && p.ReleaseYear != nil {
		album = strconv.Itoa(*p.ReleaseYear)
	}

	title := fmt.Sprintf("%s %s %s", p.Format, p.Artist, album)
	if p.IsRepeated {
		title = fmt.Sprintf("Disco Vinil %s %s", album, p.Artist)
	}

	double := ""
	if p.IsDoubleCovered {
		double = "Capa Dupla"
	}
	if p.LPsQuantity > 1 && p.LPsQuantity < 10 {
		double = QuantityTranslation[p.LPsQuantity]
	}
	if double != "" {
		title += " " + double
	}

	if p.GatefoldQuantity >= 2 {
		title += " Com Encartes"
	} else if p.GatefoldQuantity == 1 {
		title += " Com Encarte"
	}

	if p.IsRepeated {
		title += " A"
	}

	if p.IsNew {
		title += " Novo Lacrado"
	}

	if p.IsImported != nil && *p.IsImported {
		title += " Importado"
	}

	if p.Observation != nil && *p.Observation != "" {
		title += ", Leia"
	}

	title = strings.Join(strings.Fields(title), " ")

	if len(title) > MLCharacterLimit {
		title = editFunc(title)
	}

	return title
}
