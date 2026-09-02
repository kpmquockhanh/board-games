package game

import (
	"math/rand"
)

type CardInfo struct {
	Name string
	ID   string
}

var CardCategories = map[string][]CardInfo{
	"explosive": {
		{Name: "Exploding Kitten - TNT Ship", ID: "ex_001"},
		{Name: "Exploding Kitten - House Grenade", ID: "ex_002"},
		{Name: "Exploding Kitten - Nuclear Bombs", ID: "ex_003"},
		{Name: "Exploding Kitten - Warp Core", ID: "ex_004"},
		{Name: "Exploding Kitten - Alien", ID: "ex_005"},
		{Name: "Exploding Kitten - C4", ID: "ex_006"},
		{Name: "Exploding Kitten - Science", ID: "ex_007"},
		{Name: "Exploding Kitten - Playground", ID: "ex_008"},
		{Name: "Exploding Kitten - Car Off Cliff", ID: "ex_009"},
		{Name: "Exploding Kitten - Balloon Bomb", ID: "ex_010"},
	},
	"defense": {
		{Name: "Defuse - Belly Rubs", ID: "df_001"},
		{Name: "Defuse - Tummy Rubs", ID: "df_002"},
		{Name: "Defuse - Catnip Sandwiches", ID: "df_003"},
		{Name: "Defuse - Catnip Sweater", ID: "df_004"},
		{Name: "Defuse - Kitten Therapy", ID: "df_005"},
		{Name: "Defuse - Kitten Yoga", ID: "df_006"},
		{Name: "Defuse - Laser Pointer", ID: "df_007"},
		{Name: "Defuse - Laser Tag", ID: "df_008"},
		{Name: "Defuse - Nature Documentaries", ID: "df_009"},
		{Name: "Defuse - 3AM Flatulence", ID: "df_010"},
	},
	"attack": {
		{Name: "Attack", ID: "at_001"},
		{Name: "Attack", ID: "at_002"},
		{Name: "Attack", ID: "at_003"},
		{Name: "Attack", ID: "at_004"},
		{Name: "Attack", ID: "at_005"},
	},
	"skip": {
		{Name: "Skip", ID: "sk_001"},
		{Name: "Skip", ID: "sk_002"},
		{Name: "Skip", ID: "sk_003"},
		{Name: "Skip", ID: "sk_004"},
		{Name: "Skip", ID: "sk_005"},
	},
	"super_skip": {
		{Name: "Super Skip", ID: "sk_009"},
	},
	"reverse": {
		{Name: "Reverse", ID: "sk_010"},
		{Name: "Reverse", ID: "sk_011"},
	},
	"future_vision": {
		{Name: "See the Future", ID: "fv_001"},
		{Name: "See the Future", ID: "fv_002"},
		{Name: "See the Future", ID: "fv_003"},
	},
	"nope": {
		{Name: "Nope", ID: "np_002"},
		{Name: "Nope", ID: "np_003"},
		{Name: "Nope", ID: "np_004"},
		{Name: "Nope", ID: "np_005"},
		{Name: "Nope", ID: "np_006"},
	},
	"shuffle": {
		{Name: "Shuffle", ID: "dm_001"},
		{Name: "Shuffle", ID: "dm_002"},
		{Name: "Shuffle", ID: "dm_003"},
		{Name: "Shuffle", ID: "dm_004"},
		{Name: "Shuffle", ID: "dm_005"},
	},
	"draw_from_bottom": {
		{Name: "Draw From Bottom", ID: "dm_011"},
	},
	"swap_top_and_bottom": {
		{Name: "Swap Top & Bottom", ID: "dm_012"},
	},
	"garbage_collection": {
		{Name: "Garbage Collection", ID: "dm_013"},
	},
	"catomic_bomb": {
		{Name: "Catomic Bomb", ID: "dm_014"},
	},
	"mark": {
		{Name: "Mark", ID: "dm_015"},
	},
	"bury": {
		{Name: "Bury", ID: "dm_016"},
		{Name: "Bury", ID: "dm_017"},
	},
	"dig_deeper": {
		{Name: "Dig Deeper", ID: "dm_018"},
		{Name: "Dig Deeper", ID: "dm_019"},
		{Name: "Dig Deeper", ID: "dm_020"},
		{Name: "Dig Deeper", ID: "dm_021"},
	},
	"favor": {
		{Name: "Favor", ID: "sf_004"},
		{Name: "Favor", ID: "sf_005"},
		{Name: "Favor", ID: "sf_006"},
		{Name: "Favor", ID: "sf_007"},
		{Name: "Favor", ID: "sf_008"},
		{Name: "Favor", ID: "sf_009"},
	},
	"clone": {
		{Name: "Clone", ID: "sf_011"},
		{Name: "Clone", ID: "sf_012"},
		{Name: "Clone", ID: "sf_013"},
	},
	"cat_cards": {
		{Name: "Tacocat", ID: "cc_001"},
		{Name: "Rainbow Ralphing Cat", ID: "cc_002"},
		{Name: "Beard Cat", ID: "cc_003"},
		{Name: "Hairy Potato Cat", ID: "cc_004"},
		{Name: "Cattermelon", ID: "cc_005"},
	},
	"group_effects": {
		{Name: "Group Pull", ID: "cc_025"},
	},
	"special_power": {
		{Name: "Catterbox", ID: "cc_026"},
	},
}

var baseScales = map[string]int{
	"defense":             4,
	"attack":              3,
	"skip":             3,
	"super_skip":        1,
	"reverse":           2,
	"future_vision":       4,
	"nope":                3,
	"shuffle":             4,
	"draw_from_bottom":    1,
	"swap_top_and_bottom": 1,
	"garbage_collection":  1,
	"catomic_bomb":        1,
	"mark":                1,
	"bury":                1,
	"dig_deeper":          2,
	"favor":               3,
	"clone":               2,
	"group_effects":       2,
	"special_power":       2,
	"cat_cards":           4,
}

var categoryNameMap = map[string]string{
	"attack":              "Attack",
	"skip":             "Skip",
	"super_skip":        "Super Skip",
	"reverse":           "Reverse",
	"future_vision":       "Future Vision",
	"nope":                "Nope",
	"shuffle":             "Shuffle",
	"draw_from_bottom":    "Draw From Bottom",
	"swap_top_and_bottom": "Swap Top & Bottom",
	"garbage_collection":  "Garbage Collection",
	"catomic_bomb":        "Catomic Bomb",
	"mark":                "Mark",
	"bury":                "Bury",
	"dig_deeper":          "Dig Deeper",
	"favor":               "Favor",
	"clone":               "Clone",
	"group_effects":       "Group Effects",
	"special_power":       "Special Power",
	"cat_cards":           "Cat Cards",
	"explosive":           "Explosive",
	"defense":             "Defense",
}

var nopeableCategories = map[string]bool{
	"attack":              true,
	"skip":             true,
	"super_skip":        true,
	"reverse":           true,
	"future_vision":       true,
	"shuffle":             true,
	"draw_from_bottom":    true,
	"swap_top_and_bottom": true,
	"garbage_collection":  true,
	"catomic_bomb":        true,
	"mark":                true,
	"bury":                true,
	"dig_deeper":          true,
	"favor":               true,
	"clone":               true,
	"group_effects":       true,
	"special_power":       true,
}

func GetCardCategory(cardID string) string {
	for cat, cards := range CardCategories {
		for _, c := range cards {
			if c.ID == cardID {
				return cat
			}
		}
	}
	return "unknown"
}

func GetCardData(cardID string) *CardInfo {
	for _, cards := range CardCategories {
		for _, c := range cards {
			if c.ID == cardID {
				return &c
			}
		}
	}
	return nil
}

func GetCategoryName(cardID string) string {
	cat := GetCardCategory(cardID)
	if name, ok := categoryNameMap[cat]; ok {
		return name
	}
	return cat
}

func IsNopeable(category string) bool {
	return nopeableCategories[category]
}

func Shuffle(arr []string) []string {
	a := make([]string, len(arr))
	copy(a, arr)
	for i := len(a) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
	return a
}

func RandomCardFromCategory(category string) string {
	cards := CardCategories[category]
	if len(cards) == 0 {
		return ""
	}
	return cards[rand.Intn(len(cards))].ID
}

func BuildDeck(playerCount int, multiplier float64, enabledCategories map[string]bool) []string {
	deck := []string{}

	explosiveCount := playerCount - 1

	if enabledCategories == nil || enabledCategories["explosive"] != false {
		explosives := CardCategories["explosive"]
		for i := 0; i < explosiveCount; i++ {
			deck = append(deck, explosives[i%len(explosives)].ID)
		}
	}

	for cat, baseCount := range baseScales {
		if enabledCategories != nil && enabledCategories[cat] == false {
			continue
		}
		count := int(float64(baseCount) * multiplier + 0.5)
		cards := CardCategories[cat]
		for i := 0; i < count; i++ {
			deck = append(deck, cards[i%len(cards)].ID)
		}
	}

	return Shuffle(deck)
}
