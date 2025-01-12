package pipeline

import (
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"
)

var globalLogger = logrus.New() // global logger that can be overridden by user.

// SetGlobalLogger allows changing the package-wide default logger.
func SetGlobalLogger(l *logrus.Logger) {
	if l != nil {
		globalLogger = l
	}
}

// Step represents a single step in the pipeline.
type Step struct {
	Name     string
	Callable interface{}
}

// Pipeline orchestrates steps, storing overall config and outputs.
type Pipeline struct {
	steps        []Step
	context      *ExecutionContext
	config       *PipelineConfig
	logger       *logrus.Logger
	stepOutputs  map[string][]interface{}
	pickCounters map[reflect.Type]int
}

func NewPipeline(config *PipelineConfig, logger *logrus.Logger) *Pipeline {
	if config == nil {
		config = NewPipelineConfig()
	}
	if logger == nil {
		logger = globalLogger
	}
	return &Pipeline{
		steps:        []Step{},
		context:      NewExecutionContext(),
		config:       config,
		logger:       logger,
		stepOutputs:  make(map[string][]interface{}),
		pickCounters: make(map[reflect.Type]int),
	}
}

func (p *Pipeline) SetLogger(logger *logrus.Logger) {
	if logger != nil {
		p.logger = logger
	}
}

func (p *Pipeline) SetLogLevel(level logrus.Level) {
	p.logger.SetLevel(level)
}

func (p *Pipeline) AddStep(name string, callable interface{}) {
	p.steps = append(p.steps, Step{Name: name, Callable: callable})
	p.logger.Debugf("Added step %q", name)
}

func (p *Pipeline) AddInitialInputs(inputs ...interface{}) {
	p.context.AddInputs(inputs...)
	p.logger.Debugf("Added %d initial inputs", len(inputs))
}

func (p *Pipeline) Execute() (map[string][]interface{}, error) {
	// 1) Possibly reorder steps based on config.StepOrder
	p.reorderStepsIfNeeded()

	// 2) Execute steps
	for _, step := range p.steps {
		p.logger.Infof("Executing step %q", step.Name)

		// Reset pickCounters for each step
		p.pickCounters = make(map[reflect.Type]int)

		if err := p.executeStep(step); err != nil {
			p.logger.Errorf("Step %q failed: %v", step.Name, err)
			return nil, err
		}
	}

	// 3) Filter outputs if specified
	finalOutputs := p.filterOutputs()
	p.logger.Info("Pipeline execution complete")
	return finalOutputs, nil
}

// reorderStepsIfNeeded reorders p.steps according to config.StepOrder (if any).
func (p *Pipeline) reorderStepsIfNeeded() {
	if len(p.config.StepOrder) == 0 {
		// No step order specified, do nothing
		return
	}

	// Step 1: build a map from stepName => pointer to Step (for quick lookup)
	stepMap := make(map[string]*Step)
	for i := range p.steps {
		stepMap[p.steps[i].Name] = &p.steps[i]
	}

	// Step 2: build an ordered list of steps from config.StepOrder
	used := make(map[string]bool)
	var ordered []Step

	for _, desiredName := range p.config.StepOrder {
		st, exists := stepMap[desiredName]
		if !exists {
			p.logger.Warnf("Step name %q in StepOrder does not exist in pipeline steps", desiredName)
			continue
		}
		ordered = append(ordered, *st)
		used[desiredName] = true
	}

	// Step 3: add remaining steps (those not used yet) in their original order
	for _, s := range p.steps {
		if !used[s.Name] {
			ordered = append(ordered, s)
		}
	}

	// Step 4: overwrite p.steps with the new ordering
	p.steps = ordered
}

func (p *Pipeline) executeStep(step Step) error {
	fnValue := reflect.ValueOf(step.Callable)
	fnType := fnValue.Type()
	numIn := fnType.NumIn()
	args := make([]reflect.Value, numIn)

	stepCfg, hasStepCfg := p.config.StepConfigs[step.Name]
	var bindings []*ArgBinding
	if hasStepCfg {
		bindings = stepCfg.ArgBindings
	}

	for i := 0; i < numIn; i++ {
		var argVal reflect.Value
		var err error

		// If we have a custom ArgBinding, use it; else default
		if hasStepCfg && i < len(bindings) && bindings[i] != nil {
			argVal, err = p.resolveArg(step, fnType.In(i), bindings[i])
		} else {
			argVal, err = p.resolveArgDefault(step, fnType.In(i))
		}

		if err != nil {
			return err
		}
		args[i] = argVal
	}

	results := fnValue.Call(args)
	p.context.StoreResults(results)

	var resultInterfaces []interface{}
	for _, r := range results {
		resultInterfaces = append(resultInterfaces, r.Interface())
	}
	p.stepOutputs[step.Name] = append(p.stepOutputs[step.Name], resultInterfaces...)

	p.logger.Debugf("Step %q produced %d outputs", step.Name, len(results))
	return nil
}

func (p *Pipeline) resolveArg(step Step, paramType reflect.Type, binding *ArgBinding) (reflect.Value, error) {
	switch binding.Source {
	case ArgSourceInitial:
		return p.resolveArgFromInitial(step, paramType, binding.Index)
	case ArgSourceFunctionOutput:
		return p.resolveArgFromFunctionOutput(step, paramType, binding.Name, binding.Index)
	case ArgSourceDefault:
		return p.resolveArgDefault(step, paramType)
	default:
		return p.resolveArgDefault(step, paramType)
	}
}

func (p *Pipeline) resolveArgDefault(step Step, paramType reflect.Type) (reflect.Value, error) {
	switch p.config.MissingArgPolicy {
	case MissingArgPolicyUseLatest:
		idx := p.pickCounters[paramType]
		val, err := p.context.getValueByIndex(paramType, idx)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("step %s: cannot find value for type %s: %w",
				step.Name, paramType, err)
		}
		vals := p.context.values[paramType]
		if idx < len(vals)-1 {
			p.pickCounters[paramType] = idx + 1
		}
		return val, nil

	case MissingArgPolicyFail:
		return reflect.Value{}, fmt.Errorf("step %s: missing argument for type %s (policy=fail)",
			step.Name, paramType)

	default:
		return reflect.Value{}, fmt.Errorf("step %s: unknown MissingArgPolicy", step.Name)
	}
}

func (p *Pipeline) resolveArgFromInitial(step Step, paramType reflect.Type, index int) (reflect.Value, error) {
	allInitial := p.context.InitialValues()
	if index < 0 || index >= len(allInitial) {
		return reflect.Value{}, fmt.Errorf("step %s: ArgSourceInitial index %d out of range (%d total)",
			step.Name, index, len(allInitial))
	}
	val := allInitial[index]
	if !val.Type().AssignableTo(paramType) {
		return reflect.Value{}, fmt.Errorf("step %s: initial input %d has type %s, not assignable to %s",
			step.Name, index, val.Type(), paramType)
	}
	return val, nil
}

func (p *Pipeline) resolveArgFromFunctionOutput(step Step, paramType reflect.Type, funcName string, outputIndex int) (reflect.Value, error) {
	outputs, ok := p.stepOutputs[funcName]
	if !ok {
		return reflect.Value{}, fmt.Errorf("step %s: function %s has no recorded outputs", step.Name, funcName)
	}
	if outputIndex < 0 || outputIndex >= len(outputs) {
		return reflect.Value{}, fmt.Errorf("step %s: requested output index %d of function %s but it has %d outputs",
			step.Name, outputIndex, funcName, len(outputs))
	}
	out := outputs[outputIndex]
	val := reflect.ValueOf(out)
	if !val.Type().AssignableTo(paramType) {
		return reflect.Value{}, fmt.Errorf("step %s: output type %s from function %s not assignable to %s",
			step.Name, val.Type(), funcName, paramType)
	}
	return val, nil
}

func (p *Pipeline) filterOutputs() map[string][]interface{} {
	if len(p.config.OutputFilter) == 0 {
		return p.stepOutputs
	}
	selected := make(map[string][]interface{})
	filterSet := make(map[string]struct{})
	for _, name := range p.config.OutputFilter {
		filterSet[name] = struct{}{}
	}
	for stepName, outputs := range p.stepOutputs {
		if _, wanted := filterSet[stepName]; wanted {
			selected[stepName] = outputs
		}
	}
	return selected
}
