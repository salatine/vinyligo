package models

type ProductSuggestion struct {
    Format           *string
    Artist           *string
    Album            *string
    LPsQuantity      *int
    Genres           []string
    IsNew            *bool
    IsNational       *bool
    IsRepeated       *bool
    Stock            *int
    IsDoubleCovered  *bool
    SongQuantity     *int
    AlbumDuration    *float64
    ReleaseYear      *int
    Label            *string
    Observation      *string
    IsImported       *bool
}

func NewNullSuggestion() *ProductSuggestion {
    return &ProductSuggestion{
        Genres: []string{},
    }
}
