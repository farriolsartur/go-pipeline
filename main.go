package main

import (
	"fmt"
	"strings"

	"pipeline/pipeline"

	"github.com/sirupsen/logrus"
)

func main() {
	glog := logrus.New()
	glog.SetLevel(logrus.WarnLevel)
	pipeline.SetGlobalLogger(glog)

	config := pipeline.NewPipelineConfig()
	config.MissingArgPolicy = pipeline.MissingArgPolicyUseLatest

	plLogger := logrus.New()
	plLogger.SetLevel(logrus.DebugLevel)
	plLogger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	pl := pipeline.NewPipeline(config, plLogger)

	pl.AddStep("Step1", func() string {
		return "Hello from Step1!"
	})

	pl.AddStep("Step2", func(s string) int {
		fmt.Println("Step2 received:", s)
		return len(strings.TrimSpace(s))
	})

	pl.AddInitialInputs("extra input 1", "extra input 2")

	outputs, err := pl.Execute()
	if err != nil {
		fmt.Println("Pipeline failed:", err)
		return
	}

	fmt.Println("=== Pipeline Outputs ===")
	for stepName, outs := range outputs {
		fmt.Printf("%s => %v\n", stepName, outs)
	}
}
