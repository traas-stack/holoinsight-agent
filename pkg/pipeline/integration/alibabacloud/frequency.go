/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package alibabacloud

type (
	// intelligentFrequency is used to reduce frequency of pulling aliyun metrics.
	// Originally, we still have to pull metrics when there is no metrics data.
	// Using this instance, we disable pulling if empty data is fetched 'factor' times (recorded as 'down') in a row.
	// If any data is fetched, 'down' is set to 0.
	// If 'down' reach 'resetFactor', 'down' is forced to reset to 0.
	intelligentFrequency struct {
		// when 'down' reach 'factor', isDown returns true
		factor int
		// when 'down' reach 'resetFactor', 'down' is reset to 0
		resetFactor int
		// times for no data
		down              int
		realDown          bool
		temporaryRecovery bool
	}
)

// newIntelligentFrequency creates an instance of intelligentFrequency.
// resetFactor must greater than factor.
func newIntelligentFrequency(factor, resetFactor int) *intelligentFrequency {
	return &intelligentFrequency{
		factor:      factor,
		resetFactor: resetFactor,
		down:        0,
	}
}

func (f *intelligentFrequency) set(hasData bool) {
	if hasData {
		f.down = 0
		f.temporaryRecovery = false
		f.realDown = false
	} else {
		f.down++

		if f.temporaryRecovery {
			// after one try, still no data
			f.realDown = true
			f.temporaryRecovery = false
		}
		if f.down >= f.factor {
			f.realDown = true
		}

		// enter 'temporary recovery' mode
		if f.down >= f.resetFactor {
			f.down = 0
			f.realDown = false
			f.temporaryRecovery = true
		}
	}
}

func (f *intelligentFrequency) isDown() bool {
	return f.realDown
}
