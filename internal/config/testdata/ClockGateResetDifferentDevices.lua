local sq = require("sq")

sq.setclockgates({
	device = "collision-test-gates",
	gates = {
		{ subdivision = 1, channel = 1, note = 50 },
	},
})

sq.setresets({
	device = "collision-test-resets",
	partStart = { channel = 1, note = 50 },
})
