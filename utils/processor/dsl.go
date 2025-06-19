package processor

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kris-hansen/comanda/utils/config"
	"github.com/kris-hansen/comanda/utils/input"
	"github.com/kris-hansen/comanda/utils/models"
	"gopkg.in/yaml.v3"
)

// GenerateStepConfig defines the configuration for a generate step
type GenerateStepConfig struct {
	Model        interface{} `yaml:"model"`
	Action       interface{} `yaml:"action"`
	Output       string      `yaml:"output"`
	ContextFiles []string    `yaml:"context_files"`
}

// ProcessStepConfig defines the configuration for a process step
type ProcessStepConfig struct {
	WorkflowFile   string                 `yaml:"workflow_file"`
	Inputs         map[string]interface{} `yaml:"inputs"`
	CaptureOutputs []string               `yaml:"capture_outputs"`
}

// Processor handles the DSL processing pipeline
type Processor struct {
	config        *DSLConfig
	envConfig     *config.EnvConfig
	serverConfig  *config.ServerConfig // Add server config
	handler       *input.Handler
	validator     *input.Validator
	providers     map[string]models.Provider
	providerMutex sync.RWMutex // Protect providers map from concurrent access
	verbose       bool
	lastOutput    string
	spinner       *Spinner
	variables     map[string]string // Store variables from STDIN
	progress      ProgressWriter    // Progress writer for streaming updates
	runtimeDir    string            // Runtime directory for file operations
}

// isTestMode checks if the code is running in test mode
func isTestMode() bool {
	return flag.Lookup("test.v") != nil
}

// NewProcessor creates a new DSL processor
func NewProcessor(config *DSLConfig, envConfig *config.EnvConfig, serverConfig *config.ServerConfig, verbose bool, runtimeDir ...string) *Processor {
	// Default runtime directory to empty string if not provided
	rd := ""
	if len(runtimeDir) > 0 {
		rd = runtimeDir[0]
	}

	p := &Processor{
		config:       config,
		envConfig:    envConfig,
		serverConfig: serverConfig, // Store server config
		handler:      input.NewHandler(),
		validator:    input.NewValidator(nil),
		providers:    make(map[string]models.Provider),
		verbose:      verbose,
		spinner:      NewSpinner(),
		variables:    make(map[string]string),
		runtimeDir:   rd, // Store runtime directory
	}

	// Store runtime directory as-is (relative or empty)
	if rd != "" {
		p.debugf("Processor initialized with runtime directory: %s", rd)
	} else {
		p.debugf("Processor initialized without a specific runtime directory.")
	}

	// Log server configuration
	if p.serverConfig != nil {
		p.debugf("Server configuration:")
		p.debugf("- Enabled: %v", p.serverConfig.Enabled)
		p.debugf("- DataDir: %s", p.serverConfig.DataDir)
	} else {
		p.debugf("No server configuration provided")
	}

	// Disable spinner in test environments
	if isTestMode() {
		p.spinner.Disable()
	}

	p.debugf("Creating new validator with default extensions")
	return p
}

// SetProgressWriter sets the progress writer for streaming updates
func (p *Processor) SetProgressWriter(w ProgressWriter) {
	p.progress = w
	p.spinner.SetProgressWriter(w)
}

// SetLastOutput sets the last output value, useful for initializing with STDIN data
func (p *Processor) SetLastOutput(output string) {
	p.lastOutput = output
}

// LastOutput returns the last output value
func (p *Processor) LastOutput() string {
	return p.lastOutput
}

// debugf prints debug information if verbose mode is enabled
func (p *Processor) debugf(format string, args ...interface{}) {
	if p.verbose {
		fmt.Printf("[DEBUG][DSL] "+format+"\n", args...)
	}
}

// emitProgress sends a progress update if a progress writer is configured
func (p *Processor) emitProgress(msg string, step *StepInfo) {
	if p.progress != nil {
		p.progress.WriteProgress(ProgressUpdate{
			Type:    ProgressStep,
			Message: msg,
			Step:    step,
		})
	}
}

// emitProgressWithMetrics sends a progress update with performance metrics
func (p *Processor) emitProgressWithMetrics(msg string, step *StepInfo, metrics *PerformanceMetrics) {
	if p.progress != nil {
		p.progress.WriteProgress(ProgressUpdate{
			Type:               ProgressStep,
			Message:            msg,
			Step:               step,
			PerformanceMetrics: metrics,
		})
	}
}

// emitParallelProgress sends a progress update for a parallel step
func (p *Processor) emitParallelProgress(msg string, step *StepInfo, parallelID string) {
	if p.progress != nil {
		p.progress.WriteProgress(ProgressUpdate{
			Type:       ProgressParallelStep,
			Message:    msg,
			Step:       step,
			IsParallel: true,
			ParallelID: parallelID,
		})
	}
}

// emitParallelProgressWithMetrics sends a progress update for a parallel step with performance metrics
func (p *Processor) emitParallelProgressWithMetrics(msg string, step *StepInfo, parallelID string, metrics *PerformanceMetrics) {
	if p.progress != nil {
		p.progress.WriteProgress(ProgressUpdate{
			Type:               ProgressParallelStep,
			Message:            msg,
			Step:               step,
			IsParallel:         true,
			ParallelID:         parallelID,
			PerformanceMetrics: metrics,
		})
	}
}

// emitError sends an error update if a progress writer is configured
func (p *Processor) emitError(err error) {
	if p.progress != nil {
		p.progress.WriteProgress(ProgressUpdate{
			Type:  ProgressError,
			Error: err,
		})
	}
}

// parseVariableAssignment checks for "as $varname" syntax and returns the variable name
func (p *Processor) parseVariableAssignment(input string) (string, string) {
	parts := strings.Split(input, " as $")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return input, ""
}

// substituteVariables replaces variable references with their values
func (p *Processor) substituteVariables(text string) string {
	for name, value := range p.variables {
		text = strings.ReplaceAll(text, "$"+name, value)
	}
	return text
}

// validateStepConfig checks if all required fields are present in a step
func (p *Processor) validateStepConfig(stepName string, config StepConfig) error {
	var errors []string

	isGenerateStep := config.Generate != nil
	isProcessStep := config.Process != nil
	isStandardStep := !isGenerateStep && !isProcessStep && config.Type != "openai-responses" // Standard steps are not generate, process, or openai-responses
	isOpenAIResponsesStep := config.Type == "openai-responses"

	// Ensure a step is of one type only
	typeCount := 0
	if isStandardStep {
		typeCount++
	}
	if isGenerateStep {
		typeCount++
	}
	if isProcessStep {
		typeCount++
	}
	if isOpenAIResponsesStep { // This is a specific type of standard step, handled slightly differently
		// No increment here as it's a specialization of standard
	}

	if typeCount > 1 {
		errors = append(errors, "a step can only be one type: standard, generate, or process")
	}
	if isGenerateStep && (config.Input != nil || config.Model != nil || config.Action != nil || config.Output != nil) {
		// Allow Input: NA for generate steps if they don't need prior step's output
		if val, ok := config.Input.(string); !ok || val != "NA" {
			// errors = append(errors, "a 'generate' step should not contain 'input', 'model', 'action', or 'output' fields at the top level, unless Input is 'NA'")
		}
	}
	if isProcessStep && (config.Input != nil || config.Model != nil || config.Action != nil || config.Output != nil) {
		// Allow Input: NA for process steps
		if val, ok := config.Input.(string); !ok || val != "NA" {
			// errors = append(errors, "a 'process' step should not contain 'input', 'model', 'action', or 'output' fields at the top level, unless Input is 'NA'")
		}
	}

	if isStandardStep {
		if config.Input == nil {
			errors = append(errors, "input tag is required for standard steps (can be NA or empty, but the tag must be present)")
		}
		modelNames := p.NormalizeStringSlice(config.Model)
		if len(modelNames) == 0 {
			errors = append(errors, "model is required for standard steps (can be NA or a valid model name)")
		}
		actions := p.NormalizeStringSlice(config.Action)
		if len(actions) == 0 {
			errors = append(errors, "action is required for standard steps")
		}
		outputs := p.NormalizeStringSlice(config.Output)
		if len(outputs) == 0 {
			errors = append(errors, "output is required for standard steps (can be STDOUT for console output)")
		}
	} else if isOpenAIResponsesStep {
		// Validation specific to openai-responses type
		// For example, 'instructions' might be required instead of 'action'
		if config.Instructions == "" {
			// errors = append(errors, "'instructions' is required for 'openai-responses' type steps")
		}
		// Other openai-responses specific validations...
	} else if isGenerateStep {
		if config.Generate.Action == nil {
			errors = append(errors, "'action' is required within the 'generate' configuration")
		}
		if config.Generate.Output == "" {
			errors = append(errors, "'output' (filename) is required within the 'generate' configuration")
		}
	} else if isProcessStep {
		if config.Process.WorkflowFile == "" {
			errors = append(errors, "'workflow_file' is required within the 'process' configuration")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors in step '%s':\n- %s", stepName, strings.Join(errors, "\n- "))
	}

	return nil
}

// validateDependencies checks for dependencies between steps and ensures parallel steps don't depend on each other
func (p *Processor) validateDependencies() error {
	// Build a map of output files produced by each step
	outputFiles := make(map[string]string) // file -> step name

	// Track dependencies between steps
	dependencies := make(map[string][]string) // step name -> dependencies

	// First, collect outputs from parallel steps
	for groupName, steps := range p.config.ParallelSteps {
		p.debugf("Checking dependencies for parallel group: %s", groupName)

		// Track outputs within this parallel group to check for dependencies
		parallelOutputs := make(map[string]string) // file -> step name

		for _, step := range steps {
			outputs := p.NormalizeStringSlice(step.Config.Output)
			for _, output := range outputs {
				if output != "STDOUT" {
					// Check if this output is already produced by another parallel step
					if producerStep, exists := parallelOutputs[output]; exists {
						return fmt.Errorf("parallel step '%s' and '%s' both produce the same output file '%s', which creates a conflict",
							step.Name, producerStep, output)
					}

					// Add to parallel outputs map
					parallelOutputs[output] = step.Name

					// Add to global outputs map
					outputFiles[output] = step.Name
				}
			}

			// Check if this parallel step depends on outputs from other parallel steps
			inputs := p.NormalizeStringSlice(step.Config.Input)
			for _, input := range inputs {
				if input != "NA" && input != "STDIN" {
					// Check if this input is an output from another parallel step
					if producerStep, exists := parallelOutputs[input]; exists {
						return fmt.Errorf("parallel step '%s' depends on output '%s' from parallel step '%s', which is not allowed",
							step.Name, input, producerStep)
					}
				}
			}
		}
	}

	// Now check regular steps
	for _, step := range p.config.Steps {
		// Check inputs for dependencies
		inputs := p.NormalizeStringSlice(step.Config.Input)
		var stepDependencies []string

		for _, input := range inputs {
			if input != "NA" && input != "STDIN" {
				// Check if this input is an output from another step
				if producerStep, exists := outputFiles[input]; exists {
					stepDependencies = append(stepDependencies, producerStep)
				}
			}
		}

		// Store dependencies for this step
		if len(stepDependencies) > 0 {
			dependencies[step.Name] = stepDependencies
		}

		// Add this step's outputs to the map
		outputs := p.NormalizeStringSlice(step.Config.Output)
		for _, output := range outputs {
			if output != "STDOUT" {
				outputFiles[output] = step.Name
			}
		}
	}

	// Check for circular dependencies
	for stepName, deps := range dependencies {
		visited := make(map[string]bool)
		if err := p.checkCircularDependencies(stepName, deps, dependencies, visited); err != nil {
			return err
		}
	}

	return nil
}

// checkCircularDependencies performs a depth-first search to detect circular dependencies
func (p *Processor) checkCircularDependencies(
	currentStep string,
	dependencies []string,
	allDependencies map[string][]string,
	visited map[string]bool,
) error {
	if visited[currentStep] {
		return fmt.Errorf("circular dependency detected involving step '%s'", currentStep)
	}

	visited[currentStep] = true

	for _, dep := range dependencies {
		if nextDeps, exists := allDependencies[dep]; exists {
			if err := p.checkCircularDependencies(dep, nextDeps, allDependencies, visited); err != nil {
				return err
			}
		}
	}

	// Remove from visited when backtracking
	visited[currentStep] = false

	return nil
}

// Process executes the DSL processing pipeline
func (p *Processor) Process() error {
	// Check if we have any steps to process
	if len(p.config.Steps) == 0 && len(p.config.ParallelSteps) == 0 {
		err := fmt.Errorf("no steps defined in DSL configuration")
		p.debugf("Validation error: %v", err)
		p.emitError(err)
		return fmt.Errorf("validation failed: %w", err)
	}

	p.debugf("Initial validation passed: found %d sequential steps and %d parallel step groups",
		len(p.config.Steps), len(p.config.ParallelSteps))

	// First validate all steps before processing
	p.spinner.Start("Validating DSL configuration")

	// Validate sequential steps
	p.debugf("Starting sequential step validation for %d steps", len(p.config.Steps))
	for i, step := range p.config.Steps {
		p.debugf("Step %d: name=%s model=%v action=%v", i+1, step.Name, step.Config.Model, step.Config.Action)

		// Validate step configuration
		if err := p.validateStepConfig(step.Name, step.Config); err != nil {
			p.spinner.Stop()
			errMsg := fmt.Sprintf("Validation failed for step '%s': %v", step.Name, err)
			p.debugf("Step validation error: %s", errMsg)
			p.emitError(fmt.Errorf("%s", errMsg))
			return fmt.Errorf("validation error: %w", err)
		}

		// Validate model names only for standard or relevant steps
		if step.Config.Generate == nil && step.Config.Process == nil && step.Config.Type != "openai-responses" {
			modelNames := p.NormalizeStringSlice(step.Config.Model)
			p.debugf("Normalized model names for step %s: %v", step.Name, modelNames)
			if err := p.validateModel(modelNames, []string{"STDIN"}); err != nil { // STDIN is a placeholder here
				p.debugf("Model validation failed for step %s: %v", step.Name, err)
				return fmt.Errorf("model validation failed for step %s: %w", step.Name, err)
			}
		}

		p.debugf("Successfully validated step: %s", step.Name)
	}

	// Validate parallel steps
	for groupName, steps := range p.config.ParallelSteps {
		p.debugf("Starting parallel step validation for group '%s' with %d steps", groupName, len(steps))

		for i, step := range steps {
			p.debugf("Parallel step %d: name=%s model=%v action=%v", i+1, step.Name, step.Config.Model, step.Config.Action)

			// Validate step configuration
			if err := p.validateStepConfig(step.Name, step.Config); err != nil {
				p.spinner.Stop()
				errMsg := fmt.Sprintf("Validation failed for parallel step '%s': %v", step.Name, err)
				p.debugf("Parallel step validation error: %s", errMsg)
				p.emitError(fmt.Errorf("%s", errMsg))
				return fmt.Errorf("validation error: %w", err)
			}

			// Validate model names only for standard or relevant steps
			if step.Config.Generate == nil && step.Config.Process == nil && step.Config.Type != "openai-responses" {
				modelNames := p.NormalizeStringSlice(step.Config.Model)
				p.debugf("Normalized model names for parallel step %s: %v", step.Name, modelNames)
				if err := p.validateModel(modelNames, []string{"STDIN"}); err != nil { // STDIN is a placeholder
					p.debugf("Model validation failed for parallel step %s: %v", step.Name, err)
					return fmt.Errorf("model validation failed for parallel step %s: %w", step.Name, err)
				}
			}
			p.debugf("Successfully validated parallel step: %s", step.Name)
		}
	}

	// Validate dependencies between steps
	p.debugf("Validating dependencies between steps")
	if err := p.validateDependencies(); err != nil {
		p.spinner.Stop()
		errMsg := fmt.Sprintf("Dependency validation failed: %v", err)
		p.debugf("Dependency validation error: %s", errMsg)
		p.emitError(fmt.Errorf("%s", errMsg))
		return fmt.Errorf("dependency validation error: %w", err)
	}

	p.spinner.Stop()
	p.debugf("All steps validated successfully")

	// Process steps with detailed logging and error handling
	defer func() {
		if r := recover(); r != nil {
			p.debugf("Panic during step processing: %v", r)
			p.emitError(fmt.Errorf("internal error: %v", r))
		}
	}()

	// Store results from parallel steps for use in sequential steps
	parallelResults := make(map[string]string)

	// Process parallel steps first if any
	for groupName, steps := range p.config.ParallelSteps {
		p.spinner.Start(fmt.Sprintf("Processing parallel step group: %s", groupName))
		p.debugf("Starting parallel processing for group '%s' with %d steps", groupName, len(steps))

		// Create channels for collecting results and errors
		type stepResult struct {
			name   string
			output string
		}

		resultChan := make(chan stepResult, len(steps))
		errorChan := make(chan error, len(steps))

		// Use a WaitGroup to wait for all goroutines to complete
		var wg sync.WaitGroup

		// Launch a goroutine for each parallel step
		for _, step := range steps {
			wg.Add(1)

			// Create a copy of the step for the goroutine to avoid race conditions
			stepCopy := step

			go func() {
				defer wg.Done()

				p.debugf("Starting goroutine for parallel step: %s", stepCopy.Name)

				// Process the step
				response, err := p.processStep(stepCopy, true, groupName)
				if err != nil {
					p.debugf("Error in parallel step '%s': %v", stepCopy.Name, err)
					errorChan <- fmt.Errorf("error in parallel step '%s': %w", stepCopy.Name, err)
					return
				}

				// Send the result to the result channel
				resultChan <- stepResult{
					name:   stepCopy.Name,
					output: response,
				}

				p.debugf("Completed parallel step: %s", stepCopy.Name)
			}()
		}

		// Wait for all goroutines to complete in a separate goroutine
		go func() {
			wg.Wait()
			close(resultChan)
			close(errorChan)
		}()

		// Check for errors
		for err := range errorChan {
			p.spinner.Stop()
			p.emitError(err)
			return err
		}

		// Collect results
		for result := range resultChan {
			p.debugf("Collected result from parallel step: %s", result.name)
			parallelResults[result.name] = result.output
		}

		p.spinner.Stop()
		p.debugf("Completed all parallel steps in group: %s", groupName)
	}

	// Process sequential steps
	for stepIndex, step := range p.config.Steps {
		stepInfo := &StepInfo{
			Name:   step.Name,
			Model:  fmt.Sprintf("%v", step.Config.Model),
			Action: fmt.Sprintf("%v", step.Config.Action),
		}
		if step.Config.Generate != nil {
			stepInfo.Model = fmt.Sprintf("%v", step.Config.Generate.Model)
			stepInfo.Action = fmt.Sprintf("%v", step.Config.Generate.Action)
		} else if step.Config.Process != nil {
			stepInfo.Action = fmt.Sprintf("Process workflow: %s", step.Config.Process.WorkflowFile)
			stepInfo.Model = "N/A"
		}

		stepMsg := fmt.Sprintf("Processing step %d/%d: %s", stepIndex+1, len(p.config.Steps), step.Name)
		p.emitProgress(stepMsg, stepInfo)
		p.spinner.Start(stepMsg)

		// Process the step
		response, err := p.processStep(step, false, "")
		if err != nil {
			p.spinner.Stop()
			errMsg := fmt.Sprintf("Error processing step '%s': %v", step.Name, err)
			p.debugf("Step processing error: %s", errMsg)
			p.emitError(fmt.Errorf("%s", errMsg))
			return fmt.Errorf("step processing error: %w", err)
		}

		// Store the response for potential use as STDIN in next step
		p.lastOutput = response

		p.spinner.Stop()
		p.debugf("Successfully processed step: %s", step.Name)

		// Clear the handler's contents for the next step
		p.handler = input.NewHandler()
	}

	p.debugf("DSL processing completed successfully")
	return nil
}

// processStep handles the processing of a single step (used for both sequential and parallel processing)
func (p *Processor) processStep(step Step, isParallel bool, parallelID string) (string, error) {
	// Create performance metrics for this step
	metrics := &PerformanceMetrics{}
	startTime := time.Now()

	// Check if this is an openai-responses step
	if step.Config.Type == "openai-responses" {
		return p.processResponsesStep(step, isParallel, parallelID)
	}

	// Handle generate step
	if step.Config.Generate != nil {
		return p.processGenerateStep(step, isParallel, parallelID, metrics, startTime)
	}

	// Handle process step
	if step.Config.Process != nil {
		return p.processProcessStep(step, isParallel, parallelID, metrics, startTime)
	}

	// Create a new handler for this step to avoid conflicts in parallel processing
	stepHandler := input.NewHandler()
	p.handler = stepHandler

	stepInfo := &StepInfo{
		Name:   step.Name,
		Model:  fmt.Sprintf("%v", step.Config.Model),
		Action: fmt.Sprintf("%v", step.Config.Action),
	}

	// Emit progress update based on whether this is a parallel step
	if isParallel {
		stepMsg := fmt.Sprintf("Processing parallel step: %s", step.Name)
		p.emitParallelProgress(stepMsg, stepInfo, parallelID)
		p.debugf("Starting parallel step processing: name=%s", step.Name)
	} else {
		stepMsg := fmt.Sprintf("Processing step: %s", step.Name)
		p.emitProgress(stepMsg, stepInfo)
		p.debugf("Starting step processing: name=%s", step.Name)
	}

	p.debugf("Step details: input=%v model=%v action=%v output=%v",
		step.Config.Input, step.Config.Model, step.Config.Action, step.Config.Output)

	// Handle input based on type with error context
	var inputs []string
	inputStartTime := time.Now()
	p.debugf("Processing input configuration for step: %s", step.Name)
	switch v := step.Config.Input.(type) {
	case map[string]interface{}:
		// Check for database input
		if _, hasDB := v["database"]; hasDB {
			p.debugf("Processing database input for step: %s", step.Name)
			if err := p.handleDatabaseInput(v); err != nil {
				errMsg := fmt.Sprintf("Database input processing failed for step '%s': %v", step.Name, err)
				p.debugf("Database error: %s", errMsg)
				return "", fmt.Errorf("database input error: %w", err)
			}
			p.debugf("Successfully processed database input")
			// Create a temporary file with the database output
			tmpFile, err := os.CreateTemp("", "comanda-db-*.txt")
			if err != nil {
				return "", fmt.Errorf("failed to create temp file for database output: %w", err)
			}
			tmpPath := tmpFile.Name()
			defer os.Remove(tmpPath)

			if _, err := tmpFile.WriteString(p.lastOutput); err != nil {
				tmpFile.Close()
				return "", fmt.Errorf("failed to write database output to temp file: %w", err)
			}
			tmpFile.Close()

			// Set the input to the temp file path
			inputs = []string{tmpPath}
		} else if url, ok := v["url"].(string); ok {
			// Handle scraping configuration
			p.debugf("Scraping content from %s for step: %s", url, step.Name)
			if err := p.handler.ProcessScrape(url, v); err != nil {
				return "", fmt.Errorf("failed to process scraping input: %w", err)
			}
		} else {
			inputs = p.NormalizeStringSlice(step.Config.Input)
		}
	default:
		inputs = p.NormalizeStringSlice(step.Config.Input)
	}

	modelNames := p.NormalizeStringSlice(step.Config.Model)
	actions := p.NormalizeStringSlice(step.Config.Action)

	p.debugf("Step configuration:")
	p.debugf("- Inputs: %v", inputs)
	p.debugf("- Models: %v", modelNames)
	p.debugf("- Actions: %v", actions)

	// Handle STDIN specially
	if len(inputs) == 1 {
		input := inputs[0]
		if strings.HasPrefix(input, "STDIN") {
			// Initialize empty input if none provided
			if p.lastOutput == "" {
				p.lastOutput = ""
				p.debugf("No previous output available, using empty input")
			}

			// Check for variable assignment
			_, varName := p.parseVariableAssignment(input)
			if varName != "" {
				p.variables[varName] = p.lastOutput
			}

			p.debugf("Processing STDIN input for step: %s", step.Name)
			// Create a temporary file with .txt extension for the STDIN content
			tmpFile, err := os.CreateTemp("", "comanda-stdin-*.txt")
			if err != nil {
				err = fmt.Errorf("failed to create temp file for STDIN: %w", err)
				fmt.Printf("Error in step '%s': %v\n", step.Name, err)
				return "", err
			}
			tmpPath := tmpFile.Name()
			defer os.Remove(tmpPath)

			if _, err := tmpFile.WriteString(p.lastOutput); err != nil {
				tmpFile.Close()
				err = fmt.Errorf("failed to write to temp file: %w", err)
				fmt.Printf("Error in step '%s': %v\n", step.Name, err)
				return "", err
			}
			tmpFile.Close()

			// Update inputs to use the temporary file
			inputs = []string{tmpPath}
		}
	}

	// Process inputs for this step
	if len(inputs) > 0 {
		p.debugf("Processing inputs for step %s...", step.Name)
		if err := p.processInputs(inputs); err != nil {
			err = fmt.Errorf("input processing error in step %s: %w", step.Name, err)
			fmt.Printf("Error: %v\n", err)
			return "", err
		}
	}

	// Record input processing time
	metrics.InputProcessingTime = time.Since(inputStartTime).Milliseconds()
	p.debugf("Input processing completed in %d ms", metrics.InputProcessingTime)

	// Start model processing time tracking
	modelStartTime := time.Now()

	// Skip model validation and provider configuration if model is NA
	if !(len(modelNames) == 1 && modelNames[0] == "NA") {
		// Validate model for this step with detailed logging
		p.debugf("Validating models for step '%s': models=%v inputs=%v", step.Name, modelNames, inputs)
		if err := p.validateModel(modelNames, inputs); err != nil {
			errMsg := fmt.Sprintf("Model validation failed for step '%s': %v (models=%v)", step.Name, err, modelNames)
			p.debugf("Model validation error: %s", errMsg)
			return "", fmt.Errorf("model validation error: %w", err)
		}
		p.debugf("Model validation successful for step: %s", step.Name)

		// Configure providers with detailed logging
		p.debugf("Configuring providers for step '%s'", step.Name)
		if err := p.configureProviders(); err != nil {
			errMsg := fmt.Sprintf("Provider configuration failed for step '%s': %v", step.Name, err)
			p.debugf("Provider configuration error: %s", errMsg)
			return "", fmt.Errorf("provider configuration error: %w", err)
		}
		p.debugf("Provider configuration successful for step: %s", step.Name)
	}

	// Process actions with detailed logging
	p.debugf("Processing actions for step '%s'", step.Name)

	// Record model processing time
	metrics.ModelProcessingTime = time.Since(modelStartTime).Milliseconds()
	p.debugf("Model processing completed in %d ms", metrics.ModelProcessingTime)

	// Start action processing time tracking
	actionStartTime := time.Now()

	// Substitute variables in actions
	substitutedActions := make([]string, len(actions))
	for i, action := range actions {
		original := action
		substituted := p.substituteVariables(action)
		substitutedActions[i] = substituted
		if original != substituted {
			p.debugf("Variable substitution: original='%s' substituted='%s'", original, substituted)
		}
	}

	p.debugf("Executing actions: models=%v actions=%v", modelNames, substitutedActions)
	response, err := p.processActions(modelNames, substitutedActions)
	if err != nil {
		errMsg := fmt.Sprintf("Action processing failed for step '%s': %v (models=%v actions=%v)",
			step.Name, err, modelNames, substitutedActions)
		p.debugf("Action processing error: %s", errMsg)
		return "", fmt.Errorf("action processing error: %w", err)
	}
	p.debugf("Successfully processed actions for step: %s", step.Name)

	// Record action processing time
	metrics.ActionProcessingTime = time.Since(actionStartTime).Milliseconds()
	p.debugf("Action processing completed in %d ms", metrics.ActionProcessingTime)

	// Start output processing time tracking
	outputStartTime := time.Now()

	// Handle output for this step
	p.debugf("Handling output for step: %s", step.Name)

	// Handle output based on type
	var handled bool
	switch v := step.Config.Output.(type) {
	case map[string]interface{}:
		if _, hasDB := v["database"]; hasDB {
			p.debugf("Processing database output for step '%s'", step.Name)
			if err := p.handleDatabaseOutput(response, v); err != nil {
				errMsg := fmt.Sprintf("Database output processing failed for step '%s': %v (config=%v)",
					step.Name, err, v)
				p.debugf("Database output error: %s", errMsg)
				return "", fmt.Errorf("database output error: %w", err)
			}

			// For database outputs, we still want to show performance metrics in STDOUT
			if p.verbose {
				fmt.Printf("\nPerformance Metrics for step '%s':\n"+
					"- Input processing: %d ms\n"+
					"- Model processing: %d ms\n"+
					"- Action processing: %d ms\n"+
					"- Database output: (in progress)\n"+
					"- Total processing: (in progress)\n",
					step.Name,
					metrics.InputProcessingTime,
					metrics.ModelProcessingTime,
					metrics.ActionProcessingTime)
			}

			p.debugf("Successfully processed database output for step: %s", step.Name)
			handled = true
		}
	}

	// Handle regular output if not already handled
	if !handled {
		outputs := p.NormalizeStringSlice(step.Config.Output)
		p.debugf("Processing regular output for step '%s': model=%s outputs=%v",
			step.Name, modelNames[0], outputs)
		if err := p.handleOutput(modelNames[0], response, outputs, metrics); err != nil {
			errMsg := fmt.Sprintf("Output processing failed for step '%s': %v (model=%s outputs=%v)",
				step.Name, err, modelNames[0], outputs)
			p.debugf("Output processing error: %s", errMsg)
			return "", fmt.Errorf("output handling error: %w", err)
		}
		p.debugf("Successfully processed output for step: %s", step.Name)
	}

	// Record output processing time
	metrics.OutputProcessingTime = time.Since(outputStartTime).Milliseconds()
	p.debugf("Output processing completed in %d ms", metrics.OutputProcessingTime)

	// Calculate total processing time
	metrics.TotalProcessingTime = time.Since(startTime).Milliseconds()

	// Log performance metrics
	p.debugf("Step '%s' performance metrics:", step.Name)
	p.debugf("- Input processing: %d ms", metrics.InputProcessingTime)
	p.debugf("- Model processing: %d ms", metrics.ModelProcessingTime)
	p.debugf("- Action processing: %d ms", metrics.ActionProcessingTime)
	p.debugf("- Output processing: %d ms", metrics.OutputProcessingTime)
	p.debugf("- Total processing: %d ms", metrics.TotalProcessingTime)

	// Emit progress update with performance metrics
	if isParallel {
		p.emitParallelProgressWithMetrics(
			fmt.Sprintf("Completed parallel step: %s (in %d ms)", step.Name, metrics.TotalProcessingTime),
			stepInfo,
			parallelID,
			metrics)
	} else {
		p.emitProgressWithMetrics(
			fmt.Sprintf("Completed step: %s (in %d ms)", step.Name, metrics.TotalProcessingTime),
			stepInfo,
			metrics)
	}

	return response, nil
}

// processGenerateStep handles the logic for a 'generate' step
func (p *Processor) processGenerateStep(step Step, isParallel bool, parallelID string, metrics *PerformanceMetrics, startTime time.Time) (string, error) {
	stepInfo := &StepInfo{
		Name:   step.Name,
		Model:  fmt.Sprintf("%v", step.Config.Generate.Model),
		Action: fmt.Sprintf("%v", step.Config.Generate.Action),
	}
	p.debugf("Processing generate step: %s", step.Name)
	// Emit progress
	if isParallel {
		p.emitParallelProgress(fmt.Sprintf("Generating workflow: %s", step.Name), stepInfo, parallelID)
	} else {
		p.emitProgress(fmt.Sprintf("Generating workflow: %s", step.Name), stepInfo)
	}

	// 1. Determine model for generation
	var genModelName string
	if step.Config.Generate.Model != nil {
		modelNames := p.NormalizeStringSlice(step.Config.Generate.Model)
		if len(modelNames) > 0 {
			genModelName = modelNames[0] // Use the first model specified
		}
	}
	if genModelName == "" {
		genModelName = p.envConfig.DefaultGenerationModel // Use default from env config
		if genModelName == "" {
			return "", fmt.Errorf("no model specified for generate step '%s' and no default_generation_model configured", step.Name)
		}
	}
	p.debugf("Using model '%s' for workflow generation in step '%s'", genModelName, step.Name)

	// 2. Prepare the prompt for the LLM
	//    This includes the Comanda DSL guide and the user's action.
	// Use the embedded guide instead of reading from file
	dslGuide := []byte(EmbeddedLLMGuide)

	userAction := ""
	if actions := p.NormalizeStringSlice(step.Config.Generate.Action); len(actions) > 0 {
		userAction = actions[0] // Assuming single action for generation prompt
	}
	if userAction == "" {
		return "", fmt.Errorf("action for generate step '%s' is empty", step.Name)
	}

	// Handle input for generate step (e.g., from STDIN or context_files)
	var contextInput string
	if step.Config.Input != nil {
		inputValStr := fmt.Sprintf("%v", step.Config.Input)
		if inputValStr == "STDIN" {
			contextInput = p.lastOutput
			p.debugf("Generate step '%s' using STDIN content as part of prompt context.", step.Name)
		} else if inputValStr != "NA" && inputValStr != "" {
			// If Input is a file path or direct string
			inputs := p.NormalizeStringSlice(step.Config.Input)
			if len(inputs) > 0 {
				// For simplicity, concatenate all inputs if multiple are provided.
				// A more sophisticated approach might handle them differently.
				var sb strings.Builder
				for _, inputPath := range inputs {
					content, err := os.ReadFile(inputPath)
					if err != nil {
						p.debugf("Warning: could not read input file %s for generate step %s: %v", inputPath, step.Name, err)
						continue // Or handle error more strictly
					}
					sb.Write(content)
					sb.WriteString("\n\n")
				}
				contextInput = sb.String()
				p.debugf("Generate step '%s' using content from specified input files as part of prompt context.", step.Name)
			}
		}
	}

	// Add content from context_files
	var contextFilesContent strings.Builder
	for _, filePath := range step.Config.Generate.ContextFiles {
		content, err := os.ReadFile(filePath)
		if err != nil {
			p.debugf("Warning: could not read context_file %s for generate step %s: %v", filePath, step.Name, err)
			continue
		}
		contextFilesContent.Write(content)
		contextFilesContent.WriteString("\n\n")
	}

	// Get list of configured models
	configuredModels := p.envConfig.GetAllConfiguredModels()
	var modelsList string
	if len(configuredModels) > 0 {
		modelsList = fmt.Sprintf("\n--- CONFIGURED MODELS ---\nThe following models are configured and available for use:\n%s\n--- END CONFIGURED MODELS ---\n", strings.Join(configuredModels, "\n"))
	} else {
		modelsList = "\n--- CONFIGURED MODELS ---\nNo models are currently configured. Use 'NA' for model fields.\n--- END CONFIGURED MODELS ---\n"
	}

	// Create a more forceful prompt that emphasizes YAML-only output
	fullPrompt := fmt.Sprintf(`SYSTEM: You are a YAML generator. You MUST output ONLY valid YAML content. No explanations, no markdown, no code blocks, no commentary - just raw YAML.

--- BEGIN COMANDA DSL SPECIFICATION ---
%s
--- END COMANDA DSL SPECIFICATION ---
%s
User's request: %s

Additional Context (if any):
%s
%s

CRITICAL INSTRUCTION: Your entire response must be valid YAML syntax that can be directly saved to a .yaml file. Do not include ANY text before or after the YAML content. Start your response with the first line of YAML and end with the last line of YAML.

IMPORTANT: When specifying models in the generated YAML, you MUST use one of the configured models listed above, or use 'NA' if no model is needed for a step.`,
		string(dslGuide), modelsList, userAction, contextInput, contextFilesContent.String())

	// 3. Call the LLM
	provider, err := p.getProviderForModel(genModelName)
	if err != nil {
		return "", fmt.Errorf("failed to get provider for model '%s' in generate step '%s': %w", genModelName, step.Name, err)
	}

	// Create a temporary input.Input for the LLM call
	// tempLLMInput := &input.Input{ // Not needed if SendPrompt takes a string
	// 	Contents: []byte(fullPrompt),
	// 	Path:     "generate-prompt", // Placeholder path
	// 	Type:     input.StdinInput,  // Assign a type, StdinInput seems appropriate for a string prompt
	// 	MimeType: "text/plain",
	// }

	// Assuming provider is already configured via configureProviders() or similar mechanism
	generatedResponse, err := provider.SendPrompt(genModelName, fullPrompt)
	if err != nil {
		return "", fmt.Errorf("LLM execution failed for generate step '%s' with model '%s': %w", step.Name, genModelName, err)
	}

	// Extract YAML content from the response
	yamlContent := generatedResponse

	// Check if the response contains code blocks
	if strings.Contains(generatedResponse, "```yaml") {
		// Extract content between ```yaml and ```
		startMarker := "```yaml"
		endMarker := "```"

		startIdx := strings.Index(generatedResponse, startMarker)
		if startIdx != -1 {
			startIdx += len(startMarker)
			// Find the next ``` after the start marker
			remaining := generatedResponse[startIdx:]
			endIdx := strings.Index(remaining, endMarker)
			if endIdx != -1 {
				yamlContent = strings.TrimSpace(remaining[:endIdx])
			}
		}
	} else if strings.Contains(generatedResponse, "```") {
		// Try generic code block
		parts := strings.Split(generatedResponse, "```")
		if len(parts) >= 3 {
			// Take the content of the first code block
			yamlContent = strings.TrimSpace(parts[1])
			// Remove language identifier if present (e.g., "yaml" at the start)
			lines := strings.Split(yamlContent, "\n")
			if len(lines) > 0 && !strings.Contains(lines[0], ":") {
				yamlContent = strings.Join(lines[1:], "\n")
			}
		}
	}

	// 4. Validate the generated YAML before saving
	if err := p.validateGeneratedWorkflow(yamlContent); err != nil {
		return "", fmt.Errorf("generated workflow validation failed for step '%s': %w", step.Name, err)
	}

	// 5. Save the generated YAML to the output file
	outputFilePath := step.Config.Generate.Output
	if err := os.WriteFile(outputFilePath, []byte(yamlContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write generated workflow to '%s' in generate step '%s': %w", outputFilePath, step.Name, err)
	}

	p.debugf("Generated workflow saved to %s", outputFilePath)

	// Record metrics
	metrics.TotalProcessingTime = time.Since(startTime).Milliseconds()
	if isParallel {
		p.emitParallelProgressWithMetrics(fmt.Sprintf("Completed generate step: %s", step.Name), stepInfo, parallelID, metrics)
	} else {
		p.emitProgressWithMetrics(fmt.Sprintf("Completed generate step: %s", step.Name), stepInfo, metrics)
	}

	return fmt.Sprintf("Generated workflow saved to %s", outputFilePath), nil
}

// processProcessStep handles the logic for a 'process' step
func (p *Processor) processProcessStep(step Step, isParallel bool, parallelID string, metrics *PerformanceMetrics, startTime time.Time) (string, error) {
	stepInfo := &StepInfo{
		Name:   step.Name,
		Action: fmt.Sprintf("Process workflow: %s", step.Config.Process.WorkflowFile),
		Model:  "N/A",
	}
	p.debugf("Processing process step: %s, workflow_file: %s", step.Name, step.Config.Process.WorkflowFile)
	// Emit progress
	if isParallel {
		p.emitParallelProgress(fmt.Sprintf("Processing sub-workflow: %s (%s)", step.Name, step.Config.Process.WorkflowFile), stepInfo, parallelID)
	} else {
		p.emitProgress(fmt.Sprintf("Processing sub-workflow: %s (%s)", step.Name, step.Config.Process.WorkflowFile), stepInfo)
	}

	// 1. Read the sub-workflow YAML file
	subWorkflowPath := step.Config.Process.WorkflowFile
	yamlFile, err := os.ReadFile(subWorkflowPath)
	if err != nil {
		return "", fmt.Errorf("failed to read sub-workflow file '%s' for process step '%s': %w", subWorkflowPath, step.Name, err)
	}

	var subDSLConfig DSLConfig
	if err := yaml.Unmarshal(yamlFile, &subDSLConfig); err != nil {
		return "", fmt.Errorf("failed to unmarshal sub-workflow YAML '%s' for process step '%s': %w", subWorkflowPath, step.Name, err)
	}

	// 2. Create a new Processor for the sub-workflow
	//    It inherits verbose settings and envConfig, but has its own DSLConfig and variables.
	//    The runtimeDir for the sub-processor could be the directory of the sub-workflow file or inherited.
	//    For now, let's assume it inherits the parent's runtimeDir.
	subProcessor := NewProcessor(&subDSLConfig, p.envConfig, p.serverConfig, p.verbose, p.runtimeDir)
	if p.progress != nil { // Propagate progress writer if available
		subProcessor.SetProgressWriter(p.progress)
	}

	// 3. Handle inputs for the sub-workflow (optional)
	if step.Config.Process.Inputs != nil {
		for key, value := range step.Config.Process.Inputs {
			// How these inputs are made available to the sub-workflow needs careful design.
			// Option 1: Set them as initial variables in the sub-processor.
			subProcessor.variables[key] = fmt.Sprintf("%v", value) // Convert value to string
			p.debugf("Passing input '%s' (value: '%v') to sub-workflow '%s'", key, value, subWorkflowPath)

			// Option 2: If the first step of sub-workflow expects STDIN and is named,
			// we could set p.lastOutput. This is more complex.
			// For now, using variables is simpler.
		}
	}

	// If the parent 'process' step received STDIN, pass it to the sub-processor's lastOutput
	if inputValStr := fmt.Sprintf("%v", step.Config.Input); inputValStr == "STDIN" {
		subProcessor.SetLastOutput(p.lastOutput)
		p.debugf("Passing STDIN from parent step '%s' to sub-workflow '%s'", step.Name, subWorkflowPath)
	}

	// 4. Execute the sub-workflow
	if err := subProcessor.Process(); err != nil {
		return "", fmt.Errorf("error processing sub-workflow '%s' in step '%s': %w", subWorkflowPath, step.Name, err)
	}

	// 5. Handle output capture (optional)
	//    This is a placeholder for now. Capturing specific outputs from a sub-workflow
	//    and making them available to the parent requires more design (e.g., sub-workflow
	//    explicitly "exports" variables or files).
	//    For now, the main output of the sub-processor (lastOutput) could be returned.

	resultMessage := fmt.Sprintf("Successfully processed sub-workflow %s", subWorkflowPath)
	p.debugf(resultMessage)

	// Record metrics
	metrics.TotalProcessingTime = time.Since(startTime).Milliseconds()
	if isParallel {
		p.emitParallelProgressWithMetrics(fmt.Sprintf("Completed process step: %s", step.Name), stepInfo, parallelID, metrics)
	} else {
		p.emitProgressWithMetrics(fmt.Sprintf("Completed process step: %s", step.Name), stepInfo, metrics)
	}

	return subProcessor.LastOutput(), nil // Return the last output of the sub-workflow
}

// getCurrentStepConfig returns the configuration for the current step being processed
func (p *Processor) getCurrentStepConfig() StepConfig {
	// If we're not processing a step yet, return an empty config with default values
	if p.config == nil || (len(p.config.Steps) == 0 && len(p.config.ParallelSteps) == 0) {
		return StepConfig{
			BatchMode: "individual", // Default to individual mode for safety
		}
	}

	// For now, just return the first step's config
	// In a more complete implementation, this would track the current step being processed
	if len(p.config.Steps) > 0 {
		return p.config.Steps[0].Config
	}

	// If we only have parallel steps, return the first one
	for _, steps := range p.config.ParallelSteps {
		if len(steps) > 0 {
			return steps[0].Config
		}
	}

	// Fallback to default config
	return StepConfig{
		BatchMode: "individual", // Default to individual mode for safety
	}
}

// GetProcessedInputs returns all processed input contents
func (p *Processor) GetProcessedInputs() []*input.Input {
	return p.handler.GetInputs()
}

// validateGeneratedWorkflow validates that the generated YAML contains valid model references
func (p *Processor) validateGeneratedWorkflow(yamlContent string) error {
	p.debugf("Validating generated workflow YAML")

	// Parse the YAML to check for model validity
	var generatedConfig DSLConfig
	if err := yaml.Unmarshal([]byte(yamlContent), &generatedConfig); err != nil {
		return fmt.Errorf("generated YAML is invalid: %w", err)
	}

	// Collect all models referenced in the generated workflow
	var referencedModels []string

	// Check models in sequential steps
	for _, step := range generatedConfig.Steps {
		// Skip generate and process steps as they don't use models directly
		if step.Config.Generate != nil || step.Config.Process != nil {
			continue
		}

		// Check models in standard steps
		modelNames := p.NormalizeStringSlice(step.Config.Model)
		for _, modelName := range modelNames {
			if modelName != "NA" && modelName != "" {
				referencedModels = append(referencedModels, modelName)
			}
		}

		// Check models in generate steps
		if step.Config.Generate != nil && step.Config.Generate.Model != nil {
			genModelNames := p.NormalizeStringSlice(step.Config.Generate.Model)
			for _, modelName := range genModelNames {
				if modelName != "" {
					referencedModels = append(referencedModels, modelName)
				}
			}
		}
	}

	// Check models in parallel steps
	for _, steps := range generatedConfig.ParallelSteps {
		for _, step := range steps {
			// Skip generate and process steps
			if step.Config.Generate != nil || step.Config.Process != nil {
				continue
			}

			modelNames := p.NormalizeStringSlice(step.Config.Model)
			for _, modelName := range modelNames {
				if modelName != "NA" && modelName != "" {
					referencedModels = append(referencedModels, modelName)
				}
			}

			// Check models in generate steps
			if step.Config.Generate != nil && step.Config.Generate.Model != nil {
				genModelNames := p.NormalizeStringSlice(step.Config.Generate.Model)
				for _, modelName := range genModelNames {
					if modelName != "" {
						referencedModels = append(referencedModels, modelName)
					}
				}
			}
		}
	}

	// Validate each referenced model
	invalidModels := []string{}
	for _, modelName := range referencedModels {
		p.debugf("Checking if model '%s' in generated workflow is valid", modelName)

		// Check if provider exists for this model
		provider := models.DetectProvider(modelName)
		if provider == nil {
			invalidModels = append(invalidModels, fmt.Sprintf("%s (no provider found)", modelName))
			continue
		}

		// Check if provider supports this model
		if !provider.SupportsModel(modelName) {
			invalidModels = append(invalidModels, fmt.Sprintf("%s (not supported by %s)", modelName, provider.Name()))
			continue
		}

		// Check if model is configured
		_, err := p.envConfig.GetModelConfig(provider.Name(), modelName)
		if err != nil {
			if strings.Contains(err.Error(), "not found for provider") {
				invalidModels = append(invalidModels, fmt.Sprintf("%s (not configured - run 'comanda configure' to add it)", modelName))
			} else {
				invalidModels = append(invalidModels, fmt.Sprintf("%s (configuration error: %v)", modelName, err))
			}
		}
	}

	if len(invalidModels) > 0 {
		return fmt.Errorf("generated workflow contains invalid or unconfigured models:\n  - %s", strings.Join(invalidModels, "\n  - "))
	}

	p.debugf("Generated workflow validation successful - all %d model references are valid", len(referencedModels))
	return nil
}

// getProviderForModel retrieves a model provider based on the model name
// TODO: Implement actual logic to select provider based on model name and envConfig
func (p *Processor) getProviderForModel(modelName string) (models.Provider, error) {
	// For now, return the first configured provider or an error if none exist
	// This is a placeholder and needs to be replaced with actual provider selection logic
	for _, provider := range p.providers {
		// This simple logic just returns the first provider.
		// A real implementation would look up the provider that supports `modelName`.
		return provider, nil
	}
	// Fallback or more sophisticated lookup if p.providers is not populated yet
	// or if a specific provider for modelName is needed.
	// This might involve looking into p.envConfig.Providers.

	// Placeholder: Attempt to find a provider that lists this model.
	// This is a simplified lookup. A more robust system would map model names to provider types.
	for providerName, providerConfig := range p.envConfig.Providers {
		for _, model := range providerConfig.Models {
			if model.Name == modelName {
				// Found a provider that lists this model. Now get/initialize this provider.
				// This part depends on how providers are initialized and stored in p.providers.
				// If p.providers is already populated by configureProviders(), this might not be needed here.
				// For now, let's assume p.providers is populated.
				if prov, ok := p.providers[providerName]; ok {
					return prov, nil
				}
				// If not in p.providers, we might need to initialize it.
				// This is complex and depends on the overall provider management strategy.
				// For this stub, we'll return an error if not found in the initialized map.
				return nil, fmt.Errorf("provider '%s' for model '%s' not initialized in p.providers", providerName, modelName)
			}
		}
	}

	return nil, fmt.Errorf("no provider configured or found for model %s", modelName)
}
