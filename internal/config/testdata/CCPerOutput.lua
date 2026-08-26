local sq = require("sq")

sq.addinstrument({
	name = "Fallback Instrument For CC Test",
	controlchanges = {
		{ 1, 127, "Fallback Mod" },
	},
})

sq.addinstrument({
	name = "Device Specific For CC Test",
	output = "cc-test-device",
	controlchanges = {
		{ 10, 127, "Device Pan" },
		{ 20, 64, "Device Filter" },
	},
})
