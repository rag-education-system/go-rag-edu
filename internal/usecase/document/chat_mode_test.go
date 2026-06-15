package document

import "testing"

func TestParseChatMode(t *testing.T) {
	tests := []struct {
		in   string
		want ChatMode
	}{
		{"", ChatModeHybrid},
		{"hybrid", ChatModeHybrid},
		{"strict", ChatModeStrict},
		{"document_only", ChatModeStrict},
	}

	for _, tt := range tests {
		if got := ParseChatMode(tt.in); got != tt.want {
			t.Errorf("ParseChatMode(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPlanRAGResponseStrictWithoutDocs(t *testing.T) {
	plan := PlanRAGResponse(ChatModeStrict, &RAGResult{}, "apa itu sorting?")
	if plan.UseLLM {
		t.Fatal("expected no LLM for strict mode without docs")
	}
	if plan.Strategy != ResponseRefusal {
		t.Fatalf("strategy = %q, want refusal", plan.Strategy)
	}
}

func TestPlanRAGResponseHybridFactualWithoutDocs(t *testing.T) {
	plan := PlanRAGResponse(ChatModeHybrid, &RAGResult{}, "berapa SKS mata kuliah ini?")
	if plan.UseLLM {
		t.Fatal("expected refusal for factual hybrid query without docs")
	}
	if plan.Strategy != ResponseRefusal {
		t.Fatalf("strategy = %q, want refusal", plan.Strategy)
	}
}

func TestPlanRAGResponseHybridConceptualWithoutDocs(t *testing.T) {
	plan := PlanRAGResponse(ChatModeHybrid, &RAGResult{}, "apa itu algoritma sorting?")
	if !plan.UseLLM {
		t.Fatal("expected LLM for conceptual hybrid query without docs")
	}
	if plan.Strategy != ResponseGeneralKnowledge {
		t.Fatalf("strategy = %q, want general", plan.Strategy)
	}
}

func TestIsFactualQuery(t *testing.T) {
	if !IsFactualQuery("berapa SKS praktikum ini?") {
		t.Fatal("expected factual query")
	}
	if IsFactualQuery("apa itu quicksort?") {
		t.Fatal("expected conceptual query")
	}
}
