local sq = require("sq")

sq.setclockgates({
	device = "range-collision-test",
	gates = {
		{ subdivision = 3, channel = 1, note = 71 },
	},
})

sq.setresets({
	device = "range-collision-test",
	starts = { channel = 1, startnote = 70, endnote = 72 },
})
