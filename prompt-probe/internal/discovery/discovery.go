// Package discovery maps a completed probe into a GraphRAG
// DiscoveryResult. It is a pure function: no network, no daemon calls.
//
// The Gibson daemon's DiscoveryProcessor reflects on field 100 of the
// tool's response (a *graphragpb.DiscoveryResult) and writes the entries
// into the Neo4j knowledge graph automatically. This package records the
// probed Endpoint and, when a MITRE technique id is supplied, a Technique
// custom node related to that endpoint — so a campaign's graph shows
// which endpoints were exercised with which techniques.
package discovery

import (
	"google.golang.org/protobuf/proto"

	graphragpb "github.com/zeroroot-ai/sdk/api/gen/gibson/graphrag/v1"
)

// Build returns the DiscoveryResult for one probe. method and statusCode
// describe the HTTP exchange; techniqueID is an optional MITRE ATT&CK id
// (e.g. "T1190"). An empty techniqueID records only the endpoint.
func Build(targetURL, method string, statusCode int32, techniqueID string) *graphragpb.DiscoveryResult {
	endpoint := &graphragpb.Endpoint{
		Url:        targetURL,
		Method:     proto.String(method),
		StatusCode: proto.Int32(statusCode),
	}
	result := &graphragpb.DiscoveryResult{
		Endpoints: []*graphragpb.Endpoint{endpoint},
	}

	if techniqueID != "" {
		result.CustomNodes = []*graphragpb.CustomNode{{
			NodeType:     "Technique",
			IdProperties: map[string]string{"mitre_id": techniqueID},
			Properties:   map[string]string{"name": techniqueID},
		}}
		result.ExplicitRelationships = []*graphragpb.ExplicitRelationship{{
			FromType:         "endpoint",
			FromId:           map[string]string{"url": targetURL},
			ToType:           "Technique",
			ToId:             map[string]string{"mitre_id": techniqueID},
			RelationshipType: "TESTED_WITH",
		}}
	}

	return result
}
