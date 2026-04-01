package tracker

import "math/rand/v2"

var nameAdjectives = []string{
	"Amber", "Ancient", "Ashen", "Black", "Blazing", "Blind", "Blood", "Bone",
	"Brass", "Bright", "Broken", "Bronze", "Burning", "Crimson", "Crystal",
	"Cursed", "Dark", "Dead", "Deep", "Dire", "Dread", "Dusk", "Elder",
	"Ember", "Eternal", "Fallen", "Feral", "Fierce", "Fire", "Fog", "Forgotten",
	"Frozen", "Ghost", "Gilded", "Glass", "Golden", "Grim", "Half", "Hollow",
	"Holy", "Howling", "Hungry", "Ice", "Iron", "Jade", "Last", "Living",
	"Lone", "Lost", "Mad", "Marble", "Midnight", "Misty", "Molten", "Moon",
	"Night", "Obsidian", "Old", "Pale", "Phantom", "Poison", "Red", "Rotten",
	"Rusted", "Sacred", "Savage", "Scarlet", "Shadow", "Shattered", "Silent",
	"Silver", "Sleeping", "Smoke", "Sorrow", "Spectral", "Star", "Steel",
	"Stone", "Storm", "Sun", "Thorn", "Thunder", "Twilight", "Twin", "Veiled",
	"Violet", "War", "White", "Wild", "Winter", "Witch", "Wrath",
}

var nameNouns = []string{
	"Altar", "Ambush", "Barrow", "Basin", "Blade", "Bog", "Bridge", "Cairn",
	"Canyon", "Cascade", "Castle", "Cavern", "Chapel", "Chasm", "Citadel",
	"Clearing", "Cliff", "Cloister", "Copse", "Corridor", "Cove", "Creek",
	"Crossing", "Crown", "Crypt", "Dell", "Den", "Depths", "Descent", "Ditch",
	"Dome", "Dungeon", "Falls", "Fang", "Fen", "Ferry", "Fjord", "Ford",
	"Forge", "Fort", "Fountain", "Gale", "Gate", "Glade", "Glen", "Gorge",
	"Grave", "Grove", "Gulch", "Hall", "Harbor", "Haven", "Hearth", "Heath",
	"Hideout", "Hollow", "Horde", "Horn", "Isle", "Keep", "Lair", "Landing",
	"Ledge", "Marsh", "Maze", "Mesa", "Mill", "Mine", "Mire", "Monolith",
	"Moor", "Nest", "Oasis", "Outpost", "Pass", "Path", "Peak", "Pit",
	"Plains", "Plateau", "Pool", "Port", "Pyre", "Rampart", "Ravine", "Reach",
	"Reef", "Refuge", "Ridge", "Rift", "Rise", "Ruin", "Sanctum", "Scar",
	"Shelf", "Shrine", "Siege", "Spire", "Spring", "Stand", "Summit", "Swamp",
	"Temple", "Throne", "Tomb", "Tower", "Tunnel", "Vale", "Vault", "Vigil",
	"Wall", "Ward", "Watch", "Well",
}

func generateEncounterName() string {
	adj := nameAdjectives[rand.IntN(len(nameAdjectives))]
	noun := nameNouns[rand.IntN(len(nameNouns))]
	return adj + " " + noun
}
