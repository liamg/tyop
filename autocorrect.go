package main

// adjacentKeys maps keys to their QWERTY neighbours. Used to detect
// fat-finger punctuation substitutions (e.g. "don;t" → "don't").
var adjacentKeys = map[rune][]rune{
	';':  {'\''},
	'\'': {';'},
	',':  {'.', 'm'},
	'.':  {',', '/'},
	'/':  {'.'},
	'0':  {'9', 'o'},
	'1':  {'2', 'q'},
}

// buildAutocorrect returns the full autocorrect map for the given locale.
// It merges: adjacent-key expansions for common words, contractions, and
// common misspellings, with any locale-specific overrides applied last.
func buildAutocorrect(locale Locale) map[string]string {
	m := make(map[string]string)

	// Merge in order: common → locale-specific (locale wins on conflict).
	for k, v := range commonAutocorrect {
		m[k] = v
	}
	for k, v := range localeAutocorrect[locale] {
		m[k] = v
	}
	return m
}

// commonAutocorrect covers contractions and common misspellings.
// Adjacent-key ';' → '\'' fixes are handled upstream by preprocessSemicolons.
var commonAutocorrect = map[string]string{
	// First-person contractions — also handle post-semicolon forms
	"i":        "I",
	"i'm":      "I'm",
	"i've":     "I've",
	"i'll":     "I'll",
	"i'd":      "I'd",

	// Contractions (typed without apostrophe)
	"im":       "I'm",
	"dont":     "don't",
	"cant":     "can't",
	"wont":     "won't",
	"ive":      "I've",
	"wouldnt":  "wouldn't",
	"couldnt":  "couldn't",
	"shouldnt": "shouldn't",
	"isnt":     "isn't",
	"arent":    "aren't",
	"wasnt":    "wasn't",
	"werent":   "weren't",
	"didnt":    "didn't",
	"doesnt":   "doesn't",
	"hasnt":    "hasn't",
	"hadnt":    "hadn't",
	"theyre":   "they're",
	"youre":    "you're",
	"hes":      "he's",
	"shes":     "she's",
	"whos":     "who's",
	"whats":    "what's",
	"thats":    "that's",
	"theres":   "there's",
	"heres":    "here's",

	// Common transposition/misspellings
	"teh":      "the",
	"adn":      "and",
	"taht":     "that",
	"recieve":  "receive",
	"beleive":  "believe",
	"seperate": "separate",
	"definately": "definitely",
	"definatly":  "definitely",
	"occured":  "occurred",
	"occurance": "occurrence",
	"accomodate": "accommodate",
	"embarass":  "embarrass",
	"harrass":   "harass",
	"independant": "independent",
	"existance": "existence",
	"persistance": "persistence",
	"foriegn":   "foreign",
	"freind":    "friend",
	"goverment": "government",
	"grammer":   "grammar",
	"gaurd":     "guard",
	"guidence":  "guidance",
	"humourous": "humorous",
	"ignorance": "ignorance",
	"immediatly": "immediately",
	"incidently": "incidentally",
	"knowlege":  "knowledge",
	"liason":    "liaison",
	"mischevous": "mischievous",
	"momento":   "memento",
	"millenium": "millennium",
	"miniscule": "minuscule",
	"misspell":  "misspell",
	"neccessary": "necessary",
	"noticable": "noticeable",
	"occassion": "occasion",
	"paralell":  "parallel",
	"passtime":  "pastime",
	"persistant": "persistent",
	"playright": "playwright",
	"posession": "possession",
	"prefered":  "preferred",
	"privelige": "privilege",
	"pronounciation": "pronunciation",
	"publically": "publicly",
	"questionaire": "questionnaire",
	"reccomend": "recommend",
	"relevent":  "relevant",
	"religous":  "religious",
	"restaraunt": "restaurant",
	"rythm":     "rhythm",
	"schedual":  "schedule",
	"sieze":     "seize",
	"speach":    "speech",
	"succesful": "successful",
	"suprise":   "surprise",
	"tendancy":  "tendency",
	"tommorow":  "tomorrow",
	"tounge":    "tongue",
	"truely":    "truly",
	"unforseen": "unforeseen",
	"untill":    "until",
	"wierd":     "weird",
	"wellcome":  "welcome",
}

// localeAutocorrect holds locale-specific overrides (e.g. -ise vs -ize).
var localeAutocorrect = map[Locale]map[string]string{
	EnGB: {
		"color":     "colour",
		"honor":     "honour",
		"flavor":    "flavour",
		"humor":     "humour",
		"neighbor":  "neighbour",
		"rumor":     "rumour",
		"behavior":  "behaviour",
		"labor":     "labour",
		"favorite":  "favourite",
		"organize":  "organise",
		"recognize": "recognise",
		"analyze":   "analyse",
		"realize":   "realise",
		"apologize": "apologise",
		"center":    "centre",
		"theater":   "theatre",
		"meter":     "metre",
		"fiber":     "fibre",
		"gray":      "grey",
		"tire":      "tyre",
		"program":   "programme",
		"catalog":   "catalogue",
		"dialog":    "dialogue",
	},
	EnUS: {
		"colour":    "color",
		"honour":    "honor",
		"flavour":   "flavor",
		"humour":    "humor",
		"neighbour": "neighbor",
		"rumour":    "rumor",
		"behaviour": "behavior",
		"labour":    "labor",
		"favourite": "favorite",
		"organise":  "organize",
		"recognise": "recognize",
		"analyse":   "analyze",
		"realise":   "realize",
		"apologise": "apologize",
		"centre":    "center",
		"theatre":   "theater",
		"metre":     "meter",
		"fibre":     "fiber",
		"grey":      "gray",
		"tyre":      "tire",
		"programme": "program",
		"catalogue": "catalog",
		"dialogue":  "dialog",
	},
}
