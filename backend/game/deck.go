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
		{Name: "Bear-o-Dactyl", ID: "at_001"},
		{Name: "Catterwocky", ID: "at_002"},
		{Name: "Crab-a-Pult", ID: "at_003"},
		{Name: "Penguin Diarrhea", ID: "at_004"},
		{Name: "Rubber Duck Collection", ID: "at_005"},
	},
	"skip": {
		{Name: "Crab Walk", ID: "sk_001"},
		{Name: "Hypergoat", ID: "sk_002"},
		{Name: "Bunnyraptor", ID: "sk_003"},
		{Name: "Cheetah Butt", ID: "sk_004"},
		{Name: "Whale Boner Tetherball", ID: "sk_005"},
	},
	"super_skip": {
		{Name: "Super Skip", ID: "sk_009"},
	},
	"reverse": {
		{Name: "Reverse - Try Something New", ID: "sk_010"},
		{Name: "Reverse - Doctor Visit", ID: "sk_011"},
	},
	"future_vision": {
		{Name: "See Future - Unicorn Enchilada", ID: "fv_001"},
		{Name: "See Future - Special Ops Bunnies", ID: "fv_002"},
		{Name: "See Future - Mantis Shrimp", ID: "fv_003"},
	},
	"nope": {
		{Name: "Nope Sandwich", ID: "np_002"},
		{Name: "Jackanope", ID: "np_003"},
		{Name: "Narnope", ID: "np_004"},
		{Name: "Pope of Nope", ID: "np_005"},
		{Name: "Cantanope", ID: "np_006"},
	},
	"shuffle": {
		{Name: "Shuffle - Bat Farts Plague", ID: "dm_001"},
		{Name: "Shuffle - Pomeranian Storm", ID: "dm_002"},
		{Name: "Shuffle - Transdimensional Litter Box", ID: "dm_003"},
		{Name: "Shuffle - Abracrab Lincoln", ID: "dm_004"},
		{Name: "Shuffle - Kraken Upset", ID: "dm_005"},
	},
	"draw_from_bottom": {
		{Name: "Draw From Bottom", ID: "dm_011"},
	},
	"swap_top_and_bottom": {
		{Name: "Swap Top And Bottom", ID: "dm_012"},
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
		{Name: "Bury - Cat Watches", ID: "dm_016"},
		{Name: "Bury - Buried Secret", ID: "dm_017"},
	},
	"dig_deeper": {
		{Name: "Dig Deeper - Bargained For", ID: "dm_018"},
		{Name: "Dig Deeper - Devil Deal", ID: "dm_019"},
		{Name: "Dig Deeper - Zombie Strength", ID: "dm_020"},
		{Name: "Dig Deeper - Dying Garden", ID: "dm_021"},
	},
	"favor": {
		{Name: "Peanut Butter Belly Button", ID: "sf_004"},
		{Name: "Party Squirrels", ID: "sf_005"},
		{Name: "Beard Sailing", ID: "sf_006"},
		{Name: "New Palindrome", ID: "sf_007"},
		{Name: "Fall In Love", ID: "sf_008"},
		{Name: "Horsey Ride", ID: "sf_009"},
	},
	"clone": {
		{Name: "Clone - Work Out Kinks", ID: "sf_011"},
		{Name: "Clone - Voodoo Dolls", ID: "sf_012"},
		{Name: "Clone - Nosfera Twos", ID: "sf_013"},
	},
	"cat_cards": {
		{Name: "Tacocat", ID: "cc_001"},
		{Name: "Cattermelon", ID: "cc_002"},
		{Name: "Hairy Potato Cat", ID: "cc_003"},
		{Name: "Beard Cat", ID: "cc_004"},
		{Name: "Rainbow Ralphing Cat", ID: "cc_005"},
	},
}

var baseScales = map[string]int{
	"defense":             4,
	"attack":              3,
	"skip":                3,
	"super_skip":          1,
	"reverse":             2,
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
	// Cat cards only do anything in matching pairs, so there have to be enough
	// of them for a pair to reach one hand.
	"cat_cards": 8,
}

var categoryNameMap = map[string]string{
	"attack":              "Attack",
	"skip":                "Skip",
	"super_skip":          "Super Skip",
	"reverse":             "Reverse",
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
	"cat_cards":           "Cat Cards",
	"explosive":           "Explosive",
	"defense":             "Defense",
}

var nopeableCategories = map[string]bool{
	"attack":              true,
	"skip":                true,
	"super_skip":          true,
	"reverse":             true,
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

// CardLabel names a card the way players talk about it: by what it does. The
// deck holds five different Attacks with five different jokes printed on them,
// and a log line reading "played Bear-o-Dactyl" tells nobody what happened. Cat
// cards are the exception — which cat it is, is the whole point of them.
func CardLabel(cardID string) string {
	if GetCardCategory(cardID) == "cat_cards" {
		if data := GetCardData(cardID); data != nil {
			return data.Name
		}
	}
	return GetCategoryName(cardID)
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

// categoryEnabled treats a category the settings never mention as on. A missing
// key used to mean "off", which quietly built a deck with no Exploding Kittens
// in it whenever the settings arrived incomplete.
func categoryEnabled(enabledCategories map[string]bool, cat string) bool {
	if enabledCategories == nil {
		return true
	}
	on, named := enabledCategories[cat]
	return !named || on
}

// deckScale grows the deck with the table. The base counts make a 45-card pool,
// which is about right for three players once everyone has been dealt a hand;
// past that the draw pile collapsed — six players were left with eight cards,
// five of them Exploding Kittens — and the game was over in a round.
func deckScale(playerCount int) float64 {
	if playerCount <= 3 {
		return 1
	}
	return float64(playerCount) / 3
}

// BuildDeck returns a shuffled deck for playerCount players. explosiveCount of
// -1 (or less) means the usual one fewer Exploding Kitten than players.
func BuildDeck(playerCount int, multiplier float64, enabledCategories map[string]bool, explosiveCount int) []string {
	deck := []string{}

	if explosiveCount < 0 {
		explosiveCount = playerCount - 1
	}
	if explosiveCount > 0 && categoryEnabled(enabledCategories, "explosive") {
		explosives := CardCategories["explosive"]
		for i := 0; i < explosiveCount; i++ {
			deck = append(deck, explosives[i%len(explosives)].ID)
		}
	}

	scale := multiplier * deckScale(playerCount)
	for cat, baseCount := range baseScales {
		if !categoryEnabled(enabledCategories, cat) {
			continue
		}
		count := int(float64(baseCount)*scale + 0.5)
		cards := CardCategories[cat]

		// Every card in a category does the same thing, so which picture it
		// carries only matters for cat cards — and there it matters a lot,
		// since a combo needs two of the same cat. Deal those in pairs rather
		// than one of each, or the combo can never be made at all.
		perPicture := 1
		if cat == "cat_cards" {
			perPicture = 2
		}
		for i := 0; i < count; i++ {
			deck = append(deck, cards[(i/perPicture)%len(cards)].ID)
		}
	}

	return Shuffle(deck)
}
