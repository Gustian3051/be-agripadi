package workflow

import "github.com/gustian305/backend/internal/dto"


type State = dto.DiagnoseFlowState

const (
	StateIdle               = dto.DiagnoseFlowStateIdle
	StateImageAnalyzed      = dto.DiagnoseFlowStateImageAnalyzed
	StateCollectSymptoms    = dto.DiagnoseFlowStateCollectSymptoms
	StateNormalizeSymptoms  = dto.DiagnoseFlowStateNormalizeSymptoms
	StateCollectSeverity    = dto.DiagnoseFlowStateCollectSeverity
	StateCollectGrowthStage = dto.DiagnoseFlowStateCollectGrowthStage
	StateGenerateResult     = dto.DiagnoseFlowStateGenerateResult
	StateCompleted          = dto.DiagnoseFlowStateCompleted
)

func IsTerminalState(
	state State,
) bool {

	return state == StateCompleted
}

func IsValidState(
	state State,
) bool {

	switch state {

	case
		StateIdle,
		StateImageAnalyzed,
		StateCollectSymptoms,
		StateNormalizeSymptoms,
		StateCollectSeverity,
		StateCollectGrowthStage,
		StateGenerateResult,
		StateCompleted:

		return true
	}

	return false
}