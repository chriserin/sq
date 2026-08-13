local sq = require("sq")

sq.setclockgates({
	device = "collision-test",
	gates = {
		{ subdivision = 1, channel = 1, note = 50 },
	},
})

sq.setresets({
	device = "collision-test",
	partStart = { channel = 1, note = 50 },
})
