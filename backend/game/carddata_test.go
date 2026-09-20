package game

import (
	"encoding/json"
	"os"
	"testing"
)

// The client renders cards from frontend/src/data/cards.json — the picture, the
// name, the category it groups them under. This table and that file describe
// the same cards, so when they drift the log says one thing while the card on
// screen says another, and combos the client offers get refused by the server.
func TestCardTableAgreesWithTheClientCardData(t *testing.T) {
	const path = "../../frontend/src/data/cards.json"

	blob, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("client card data not available (%v)", err)
	}

	var data struct {
		Categories map[string]struct {
			Cards []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				File string `json:"file"`
			} `json:"cards"`
		} `json:"categories"`
	}
	if err := json.Unmarshal(blob, &data); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}

	type card struct{ category, name string }
	client := map[string]card{}
	for category, group := range data.Categories {
		for _, c := range group.Cards {
			client[c.ID] = card{category, c.Name}
		}
		if _, known := CardCategories[category]; !known {
			t.Errorf("category %q exists in the client data but not in the deck", category)
		}
	}

	for category, cards := range CardCategories {
		for _, c := range cards {
			their, ok := client[c.ID]
			if !ok {
				t.Errorf("%s (%s) is dealt but the client cannot render it", c.ID, c.Name)
				continue
			}
			if their.category != category {
				t.Errorf("%s: category %q here, %q in the client data", c.ID, category, their.category)
			}
			if their.name != c.Name {
				t.Errorf("%s: named %q here, %q in the client data", c.ID, c.Name, their.name)
			}
		}
	}
}

func TestEveryDealtCategoryCanBeScaledAndNamed(t *testing.T) {
	for category := range CardCategories {
		if category == "explosive" {
			continue // dealt by player count, not by baseScales
		}
		if _, ok := baseScales[category]; !ok {
			t.Errorf("category %q is never dealt: it has no entry in baseScales", category)
		}
		if _, ok := categoryNameMap[category]; !ok {
			t.Errorf("category %q has no display name", category)
		}
	}
	for category := range baseScales {
		if _, ok := CardCategories[category]; !ok {
			t.Errorf("baseScales deals %q, which holds no cards", category)
		}
	}
}
