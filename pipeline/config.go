package pipeline

type MissingArgPolicy int

const (
	MissingArgPolicyUseLatest MissingArgPolicy = iota
	MissingArgPolicyFail
)

type ArgSourceType int

const (
	ArgSourceDefault ArgSourceType = iota
	ArgSourceInitial
	ArgSourceFunctionOutput
)

type ArgBinding struct {
	Source ArgSourceType
	Name   string // Step name if Source = ArgSourceFunctionOutput.
	Index  int    // Index in the initial inputs or in a function’s outputs.
}

type StepConfig struct {
	ArgBindings []*ArgBinding
}

type PipelineConfig struct {
	// StepOrder is a list of step names indicating the desired order.
	// Steps not listed appear afterward in their original order.
	StepOrder []string

	MissingArgPolicy MissingArgPolicy
	OutputFilter     []string
	StepConfigs      map[string]*StepConfig
}

func NewPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		StepOrder:        nil, // by default no reordering
		MissingArgPolicy: MissingArgPolicyUseLatest,
		OutputFilter:     nil, // Return outputs for all steps.
		StepConfigs:      make(map[string]*StepConfig),
	}
}
