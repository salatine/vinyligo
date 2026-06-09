package shopify

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/salatine/vinyligo/internal/models"
)

var TagRelations = map[string]map[string]string{
	"Lp Vinil": {
		"all": "LPs", "mpb": "LPs - MPB", "metal": "LPs - Metal",
		"blues": "LPs - Blues & Jazz", "jazz": "LPs - Blues & Jazz",
		"dance": "LPs - Dance Music", "trilhas": "LPs - Trilhas Sonoras",
		"rock-nacional": "LPs - Rock/POP Nacional", "pop-nacional": "LPs - Rock/POP Nacional",
		"rock-internacional": "LPs - Rock/POP Internacional", "pop-internacional": "LPs - Rock/POP Internacional",
		"samba": "LPs - Samba & Pagode", "pagode": "LPs - Samba & Pagode",
		"sertanejo": "LPs - Sertanejo",
		"classica": "LPs - Clássicas e Orquestras", "orquestra": "LPs - Clássicas e Orquestras",
		"latin": "LPs - Latinas e Europeias",
		"outros": "LPs - Outros", "outros-nacional": "LPs - Outros Nacional", "outros-internacional": "LPs - Outros Internacional",
	},
	"Compacto Vinil": {
		"all": "Compactos", "mpb": "Compactos - MPB", "metal": "Compactos - Metal",
		"blues": "Compactos - Blues & Jazz", "jazz": "Compactos - Blues & Jazz",
		"rock-nacional": "Compactos - Rock/POP Nacional", "pop-nacional": "Compactos - Rock/POP Nacional",
		"rock-internacional": "Compactos - Rock/POP Internacional", "pop-internacional": "Compactos - Rock/POP Internacional",
		"outros": "Compactos - Outros", "outros-nacional": "Compactos - Outros Nacional", "outros-internacional": "Compactos - Outros Internacional",
	},
	"CD": {
		"all": "CDs", "mpb": "CDs - MPB", "metal": "CDs - Metal",
		"blues": "CDs - Blues & Jazz", "jazz": "CDs - Blues & Jazz",
		"dance": "CDs - Dance Music", "trilhas": "CDs - Trilhas Sonoras",
		"rock-nacional": "CDs - Rock/POP Nacional", "pop-nacional": "CDs - Rock/POP Nacional",
		"rock-internacional": "CDs - Rock/POP Internacional", "pop-internacional": "CDs - Rock/POP Internacional",
		"outros": "CDs - Outros", "outros-nacional": "CDs - Outros Nacional", "outros-internacional": "CDs - Outros Internacional",
	},
	"DVD":             {"all": "DVDs"},
	"Fita K7 Cassete": {"all": "Fitas K7"},
	"LD LaserDisc":    {"all": "LDs LaserDisc"},
}

type GraphQLResponse struct {
	Data struct {
		ProductCreate struct {
			Product struct {
				ID string `json:"id"`
			} `json:"product"`
			UserErrors []struct {
				Field   []string `json:"field"`
				Message string   `json:"message"`
			} `json:"userErrors"`
		} `json:"productCreate"`
	} `json:"data"`
}

type VariantQueryResponse struct {
	Data struct {
		Product struct {
			Variants struct {
				Nodes []struct {
					ID            string `json:"id"`
					InventoryItem struct {
						ID string `json:"id"`
					} `json:"inventoryItem"`
				} `json:"nodes"`
			} `json:"variants"`
		} `json:"product"`
	} `json:"data"`
}

const productCreateMutation = `
mutation ProductCreate($product: ProductCreateInput!) {
	productCreate(product: $product) {
		product { id }
		userErrors { field message }
	}
}
`

const productCreateMediaMutation = `
mutation ProductCreateMedia($productId: ID!, $media: [CreateMediaInput!]!) {
	productCreateMedia(productId: $productId, media: $media) {
		media { id }
		mediaUserErrors { field message }
	}
}
`

const productVariantsQuery = `
query ProductVariants($id: ID!) {
	product(id: $id) {
		variants(first: 1) {
			nodes { id inventoryItem { id } }
		}
	}
}
`

const productVariantsBulkUpdateMutation = `
mutation ProductVariantsBulkUpdate($productId: ID!, $variants: [ProductVariantsBulkInput!]!) {
	productVariantsBulkUpdate(productId: $productId, variants: $variants) {
		productVariants { id }
		userErrors { field message }
	}
}
`

const inventorySetQuantitiesMutation = `
mutation InventorySetQuantities($input: InventorySetQuantitiesInput!) {
	inventorySetQuantities(input: $input) {
		inventoryAdjustmentGroup { createdAt }
		userErrors { field message code }
	}
}
`

const publishablePublishMutation = `
mutation PublishablePublish($id: ID!, $input: [PublicationInput!]!) {
	publishablePublish(id: $id, input: $input) {
		userErrors { field message }
	}
}
`

var ProductGramsRelations = map[string]float64{
	"Lp Vinil": 100000, "LD LaserDisc": 100000,
	"Compacto Vinil": 10000, "Fita K7 Cassete": 1000,
	"CD": 100, "DVD": 100,
}

var ProductCategoryRelations = map[string]string{
	"Lp Vinil":        "gid://shopify/TaxonomyCategory/me-3-4",
	"Compacto Vinil":  "gid://shopify/TaxonomyCategory/me-3-6",
	"CD":              "gid://shopify/TaxonomyCategory/me-3-3",
	"DVD":             "gid://shopify/TaxonomyCategory/me-7-3",
	"Fita K7 Cassete": "gid://shopify/TaxonomyCategory/me-3-2",
	"LD LaserDisc":    "gid://shopify/TaxonomyCategory/me-3-4",
}

func (c *Client) PublishProduct(product *models.Product, titleEditor func(string) string, pictureUploader func(string) (string, error)) (string, error) {
	pictureURLs, err := product.GetPictureURLs(pictureUploader)
	if err != nil {
		return "", err
	}

	variables := map[string]interface{}{
		"product": map[string]interface{}{
			"title":           product.Title(titleEditor),
			"descriptionHtml": strings.ReplaceAll(product.Description(), "\n", "<br/>"),
			"vendor":          "Searom Discos",
			"productType":     product.Format,
			"tags":            getProductTags(product),
			"category":        ProductCategoryRelations[product.Format],
		},
	}

	respBody, err := c.GraphQL(productCreateMutation, variables)
	if err != nil {
		return "", err
	}

	var response GraphQLResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", err
	}

	if len(response.Data.ProductCreate.UserErrors) > 0 {
		return "", fmt.Errorf("shopify: %s", response.Data.ProductCreate.UserErrors[0].Message)
	}

	productID := response.Data.ProductCreate.Product.ID

	var media []map[string]interface{}
	for _, url := range pictureURLs {
		media = append(media, map[string]interface{}{
			"mediaContentType": "IMAGE",
			"originalSource":   url,
		})
	}

	_, err = c.GraphQL(productCreateMediaMutation, map[string]interface{}{
		"productId": productID,
		"media":     media,
	})
	if err != nil {
		return "", err
	}

	var variantResp VariantQueryResponse
	variantRespBody, err := c.GraphQL(productVariantsQuery, map[string]interface{}{"id": productID})
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(variantRespBody, &variantResp); err != nil {
		return "", err
	}

	if len(variantResp.Data.Product.Variants.Nodes) == 0 {
		return "", fmt.Errorf("shopify: nenhuma variante retornada")
	}

	variant := variantResp.Data.Product.Variants.Nodes[0]

	_, err = c.GraphQL(productVariantsBulkUpdateMutation, map[string]interface{}{
		"productId": productID,
		"variants": []map[string]interface{}{{
			"id":              variant.ID,
			"price":           fmt.Sprintf("%.2f", product.Price),
			"inventoryPolicy": "DENY",
			"inventoryItem": map[string]interface{}{
				"tracked":          true,
				"requiresShipping": true,
				"measurement": map[string]interface{}{
					"weight": map[string]interface{}{
						"value": ProductGramsRelations[product.Format],
						"unit":  "GRAMS",
					},
				},
			},
		}},
	})
	if err != nil {
		return "", err
	}

	_, err = c.GraphQL(inventorySetQuantitiesMutation, map[string]interface{}{
		"input": map[string]interface{}{
			"name":                  "available",
			"reason":                "correction",
			"ignoreCompareQuantity": true,
			"quantities": []map[string]interface{}{{
				"inventoryItemId": variant.InventoryItem.ID,
				"locationId":      c.locationID,
				"quantity":        max(product.Stock, 1),
				"compareQuantity":  nil,
			}},
		},
	})
	if err != nil {
		return "", fmt.Errorf("inventory error: %w", err)
	}

	_, err = c.GraphQL(publishablePublishMutation, map[string]interface{}{
		"id": productID,
		"input": []map[string]interface{}{{
			"publicationId": c.publicationID,
		}},
	})
	if err != nil {
		return "", fmt.Errorf("publish error: %w", err)
	}

	return productID, nil
}

func getProductTags(product *models.Product) []string {
	nationality := "internacional"
	if product.IsNational {
		nationality = "nacional"
	}

	var genres []string
	for _, genre := range product.Genres {
		genres = append(genres, strings.ToLower(genre))
	}
	if product.Format == "Compacto Vinil" {
		genres = append(genres, "compactos")
	}

	var tags []string
	relations := TagRelations[product.Format]

	if tag, ok := relations["all"]; ok {
		tags = append(tags, tag)
	}
	if tag, ok := relations[nationality]; ok {
		tags = append(tags, tag)
	}
	if product.Format == "Lp Vinil" && product.IsNew {
		if tag, ok := relations["novo"]; ok {
			tags = append(tags, tag)
		}
	}

	for _, genre := range genres {
		if tag, ok := relations[genre]; ok {
			tags = append(tags, tag)
		}
		if tag, ok := relations[genre+"-"+nationality]; ok {
			tags = append(tags, tag)
		}
	}

	return tags
}
