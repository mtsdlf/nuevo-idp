package workflow

import (
	"testing"

	"go.temporal.io/sdk/testsuite"
)

func TestApplicationActivation_HappyPath(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	fake := &fakeApplicationOnboardingPort{}
	SetApplicationOnboardingPort(fake)

	env.RegisterWorkflow(ApplicationActivation)
	env.RegisterActivity(TransitionApplicationToActive)

	input := ApplicationActivationInput{ApplicationID: "app-1"}
	env.ExecuteWorkflow(ApplicationActivation, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("workflow not completed")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if fake.applicationActivationCalls != 1 {
		t.Fatalf("expected 1 application activation call, got %d", fake.applicationActivationCalls)
	}
}

func TestApplicationActivation_RequiresID(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(ApplicationActivation)

	input := ApplicationActivationInput{}
	env.ExecuteWorkflow(ApplicationActivation, input)

	if err := env.GetWorkflowError(); err == nil {
		t.Fatalf("expected error for missing ApplicationID, got nil")
	}
}
