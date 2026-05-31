package discovery_test

import (
	"testing"

	"github.com/zeroroot-ai/gibson-redteam-example/prompt-probe/internal/discovery"
)

func TestBuild_endpointOnly(t *testing.T) {
	got := discovery.Build("https://llm.example/v1/chat", "POST", 200, "")

	if len(got.GetEndpoints()) != 1 {
		t.Fatalf("want 1 endpoint, got %d", len(got.GetEndpoints()))
	}
	ep := got.GetEndpoints()[0]
	if ep.GetUrl() != "https://llm.example/v1/chat" {
		t.Errorf("url = %q", ep.GetUrl())
	}
	if ep.GetMethod() != "POST" {
		t.Errorf("method = %q", ep.GetMethod())
	}
	if ep.GetStatusCode() != 200 {
		t.Errorf("status = %d", ep.GetStatusCode())
	}
	if len(got.GetCustomNodes()) != 0 || len(got.GetExplicitRelationships()) != 0 {
		t.Errorf("no technique expected, got %d nodes / %d rels", len(got.GetCustomNodes()), len(got.GetExplicitRelationships()))
	}
}

func TestBuild_withTechnique(t *testing.T) {
	got := discovery.Build("https://llm.example/v1/chat", "POST", 200, "T1190")

	if len(got.GetCustomNodes()) != 1 {
		t.Fatalf("want 1 technique custom node, got %d", len(got.GetCustomNodes()))
	}
	cn := got.GetCustomNodes()[0]
	if cn.GetNodeType() != "Technique" {
		t.Errorf("node type = %q", cn.GetNodeType())
	}
	if cn.GetIdProperties()["mitre_id"] != "T1190" {
		t.Errorf("mitre_id = %q", cn.GetIdProperties()["mitre_id"])
	}

	if len(got.GetExplicitRelationships()) != 1 {
		t.Fatalf("want 1 relationship, got %d", len(got.GetExplicitRelationships()))
	}
	rel := got.GetExplicitRelationships()[0]
	if rel.GetRelationshipType() != "TESTED_WITH" {
		t.Errorf("rel type = %q", rel.GetRelationshipType())
	}
	if rel.GetFromId()["url"] != "https://llm.example/v1/chat" {
		t.Errorf("rel from url = %q", rel.GetFromId()["url"])
	}
	if rel.GetToId()["mitre_id"] != "T1190" {
		t.Errorf("rel to mitre_id = %q", rel.GetToId()["mitre_id"])
	}
}
