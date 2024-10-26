package processor

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeStringSlice(t *testing.T) {
	processor := NewProcessor(&DSLConfig{}, createTestEnvConfig(), false)

	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: []string{},
		},
		{
			name:     "single string",
			input:    "test",
			expected: []string{"test"},
		},
		{
			name:     "string slice",
			input:    []string{"test1", "test2"},
			expected: []string{"test1", "test2"},
		},
		{
			name:     "interface slice",
			input:    []interface{}{"test1", "test2"},
			expected: []string{"test1", "test2"},
		},
		{
			name:     "empty interface slice",
			input:    []interface{}{},
			expected: []string{},
		},
		{
			name:     "mixed type interface slice - only strings extracted",
			input:    []interface{}{"test1", 42, "test2"},
			expected: []string{"test1", "", "test2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.NormalizeStringSlice(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("NormalizeStringSlice() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewProcessor(t *testing.T) {
	config := &DSLConfig{}
	envConfig := createTestEnvConfig()
	verbose := true

	processor := NewProcessor(config, envConfig, verbose)

	if processor == nil {
		t.Error("NewProcessor() returned nil")
	}

	if processor.config != config {
		t.Error("NewProcessor() did not set config correctly")
	}

	if processor.envConfig != envConfig {
		t.Error("NewProcessor() did not set envConfig correctly")
	}

	if processor.verbose != verbose {
		t.Error("NewProcessor() did not set verbose correctly")
	}

	if processor.handler == nil {
		t.Error("NewProcessor() did not initialize handler")
	}

	if processor.validator == nil {
		t.Error("NewProcessor() did not initialize validator")
	}

	if processor.providers == nil {
		t.Error("NewProcessor() did not initialize providers map")
	}
}

func TestValidateStepConfig(t *testing.T) {
	processor := NewProcessor(&DSLConfig{}, createTestEnvConfig(), false)

	tests := []struct {
		name          string
		stepName      string
		config        StepConfig
		expectedError string
	}{
		{
			name:     "valid config",
			stepName: "test_step",
			config: StepConfig{
				Input:  "test.txt",
				Model:  "gpt-4",
				Action: "analyze",
				Output: "STDOUT",
			},
			expectedError: "",
		},
		{
			name:     "missing input tag",
			stepName: "test_step",
			config: StepConfig{
				Model:  "gpt-4",
				Action: "analyze",
				Output: "STDOUT",
			},
			expectedError: "input tag is required",
		},
		{
			name:     "missing model",
			stepName: "test_step",
			config: StepConfig{
				Input:  "test.txt",
				Action: "analyze",
				Output: "STDOUT",
			},
			expectedError: "model is required",
		},
		{
			name:     "missing action",
			stepName: "test_step",
			config: StepConfig{
				Input:  "test.txt",
				Model:  "gpt-4",
				Output: "STDOUT",
			},
			expectedError: "action is required",
		},
		{
			name:     "missing output",
			stepName: "test_step",
			config: StepConfig{
				Input:  "test.txt",
				Model:  "gpt-4",
				Action: "analyze",
			},
			expectedError: "output is required",
		},
		{
			name:     "empty input allowed",
			stepName: "test_step",
			config: StepConfig{
				Input:  "",
				Model:  "gpt-4",
				Action: "analyze",
				Output: "STDOUT",
			},
			expectedError: "",
		},
		{
			name:     "NA input allowed",
			stepName: "test_step",
			config: StepConfig{
				Input:  "NA",
				Model:  "gpt-4",
				Action: "analyze",
				Output: "STDOUT",
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processor.validateStepConfig(tt.stepName, tt.config)
			if tt.expectedError == "" {
				if err != nil {
					t.Errorf("validateStepConfig() returned unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("validateStepConfig() expected error but got none")
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("validateStepConfig() error = %v, want error containing %v", err, tt.expectedError)
				}
			}
		})
	}
}

func TestProcess(t *testing.T) {
	tests := []struct {
		name        string
		config      DSLConfig
		expectError bool
	}{
		{
			name:        "empty config",
			config:      DSLConfig{},
			expectError: true,
		},
		{
			name: "single step with missing model",
			config: DSLConfig{
				"step_one": StepConfig{
					Action: []string{"test action"},
					Output: []string{"STDOUT"},
				},
			},
			expectError: true,
		},
		{
			name: "valid single step",
			config: DSLConfig{
				"step_one": StepConfig{
					Input:  []string{"NA"},
					Model:  []string{"gpt-4"},
					Action: []string{"test action"},
					Output: []string{"STDOUT"},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewProcessor(&tt.config, createTestEnvConfig(), false)
			err := processor.Process()

			if tt.expectError && err == nil {
				t.Error("Process() expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Process() unexpected error: %v", err)
			}
		})
	}
}

func TestDebugf(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
	}{
		{
			name:    "verbose mode enabled",
			verbose: true,
		},
		{
			name:    "verbose mode disabled",
			verbose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewProcessor(&DSLConfig{}, createTestEnvConfig(), tt.verbose)
			// Note: This test only verifies that debugf doesn't panic
			// In a real scenario, you might want to capture stdout and verify the output
			processor.debugf("test message %s", "arg")
		})
	}
}
