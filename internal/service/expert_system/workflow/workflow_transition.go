package workflow

import "errors"

//
// ============================================================
// TRANSITION MAP
// ============================================================
//

var transitions = map[State][]State{

	StateIdle: {
		StateImageAnalyzed,
	},

	StateImageAnalyzed: {
		StateCollectSymptoms,
	},

	StateCollectSymptoms: {
		StateNormalizeSymptoms,
	},

	StateNormalizeSymptoms: {
		StateCollectSeverity,
	},

	StateCollectSeverity: {
		StateCollectGrowthStage,
	},

	StateCollectGrowthStage: {
		StateGenerateResult,
	},

	StateGenerateResult: {
		StateCompleted,
	},
}

func CanTransition(from State, to State) bool {

	nextStates, ok := transitions[from]

	if !ok {
		return false
	}

	for _, next := range nextStates {

		if next == to {
			return true
		}
	}

	return false
}

func NextState(current State) (State, error) {

	nextStates, ok := transitions[current]

	if !ok {

		return "",
			errors.New(
				"invalid workflow state",
			)
	}

	if len(nextStates) == 0 {

		return "",
			errors.New(
				"no next state available",
			)
	}

	return nextStates[0], nil
}

func ValidateTransition(current State, next State) error {

	if !IsValidState(current) {

		return errors.New(
			"invalid current state",
		)
	}

	if !IsValidState(next) {

		return errors.New(
			"invalid next state",
		)
	}

	if !CanTransition(current, next) {

		return errors.New(
			"workflow transition not allowed",
		)
	}

	return nil
}
