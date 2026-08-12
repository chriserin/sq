local sq = require("sq")

sq.setclockgates({
	device = "gategrid-test",
	gates = {
		{ subdivision = 1, channel = 1, note = 1 },
		{ subdivision = 2, channel = 1, note = 2 },
		{ subdivision = 2, channel = 2, note = 9 },
	},
})
