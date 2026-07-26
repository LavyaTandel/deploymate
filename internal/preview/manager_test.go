package preview

import (
	"context"
	"testing"
)

func TestCreatePreview(t *testing.T) {
	mgr := NewManager()

	req := PreviewRequest{
		RepoURL:  "https://github.com/example/repo.git",
		PRNumber: 42,
		Branch:   "feature/new-api",
		Service:  "api",
	}

	env, err := mgr.CreatePreview(context.Background(), req)
	if err != nil {
		t.Fatalf("CreatePreview() error = %v", err)
	}
	if env.ID != "api-pr-42" {
		t.Errorf("ID = %q, want %q", env.ID, "api-pr-42")
	}
	if env.PRNumber != 42 {
		t.Errorf("PRNumber = %d, want 42", env.PRNumber)
	}
	if env.Status != StatusCreating {
		t.Errorf("Status = %q, want %q", env.Status, StatusCreating)
	}
}

func TestCreatePreview_Duplicate(t *testing.T) {
	mgr := NewManager()

	req := PreviewRequest{
		PRNumber: 42,
		Service:  "api",
	}

	_, err := mgr.CreatePreview(context.Background(), req)
	if err != nil {
		t.Fatalf("First CreatePreview() error = %v", err)
	}

	_, err = mgr.CreatePreview(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for duplicate preview")
	}
}

func TestDestroyPreview(t *testing.T) {
	mgr := NewManager()

	req := PreviewRequest{PRNumber: 42, Service: "api"}
	env, _ := mgr.CreatePreview(context.Background(), req)

	err := mgr.DestroyPreview(context.Background(), env.ID)
	if err != nil {
		t.Fatalf("DestroyPreview() error = %v", err)
	}
}

func TestDestroyPreview_NotFound(t *testing.T) {
	mgr := NewManager()

	err := mgr.DestroyPreview(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Expected error for missing preview")
	}
}

func TestListPreviews(t *testing.T) {
	mgr := NewManager()

	mgr.CreatePreview(context.Background(), PreviewRequest{PRNumber: 1, Service: "api"})
	mgr.CreatePreview(context.Background(), PreviewRequest{PRNumber: 2, Service: "api"})
	mgr.CreatePreview(context.Background(), PreviewRequest{PRNumber: 3, Service: "web"})

	envs, err := mgr.ListPreviews(context.Background(), "api")
	if err != nil {
		t.Fatalf("ListPreviews() error = %v", err)
	}
	if len(envs) != 2 {
		t.Errorf("Expected 2 previews for api, got %d", len(envs))
	}
}

func TestListPreviews_All(t *testing.T) {
	mgr := NewManager()

	mgr.CreatePreview(context.Background(), PreviewRequest{PRNumber: 1, Service: "api"})
	mgr.CreatePreview(context.Background(), PreviewRequest{PRNumber: 2, Service: "web"})

	envs, err := mgr.ListPreviews(context.Background(), "")
	if err != nil {
		t.Fatalf("ListPreviews() error = %v", err)
	}
	if len(envs) != 2 {
		t.Errorf("Expected 2 total previews, got %d", len(envs))
	}
}
