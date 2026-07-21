package repository

import (
	"math"
	"testing"
)

// ===== cosineSimilarity =====

func TestCosineSimilarity_IdenticalVectors(t *testing.T) {
	a := []float64{1, 2, 3, 4}
	score := cosineSimilarity(a, a)
	if math.Abs(score-1.0) > 1e-9 {
		t.Errorf("Identical vectors should have cosine similarity 1.0, got %f", score)
	}
}

func TestCosineSimilarity_OrthogonalVectors(t *testing.T) {
	a := []float64{1, 0}
	b := []float64{0, 1}
	score := cosineSimilarity(a, b)
	if math.Abs(score) > 1e-9 {
		t.Errorf("Orthogonal vectors should have cosine similarity 0, got %f", score)
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	zero := []float64{0, 0, 0}
	nonzero := []float64{1, 2, 3}
	score := cosineSimilarity(zero, nonzero)
	if score != 0 {
		t.Errorf("Zero vector should have cosine similarity 0, got %f", score)
	}
}

func TestCosineSimilarity_ScaledVectors(t *testing.T) {
	a := []float64{1, 2, 3}
	b := []float64{2, 4, 6} // same direction, scaled by 2
	score := cosineSimilarity(a, b)
	if math.Abs(score-1.0) > 1e-9 {
		t.Errorf("Scaled vectors in same direction should have cosine similarity 1.0, got %f", score)
	}
}

func TestCosineSimilarity_OppositeVectors(t *testing.T) {
	a := []float64{1, 2, 3}
	b := []float64{-1, -2, -3}
	score := cosineSimilarity(a, b)
	if math.Abs(score-(-1.0)) > 1e-9 {
		t.Errorf("Opposite vectors should have cosine similarity -1.0, got %f", score)
	}
}

func TestCosineSimilarity_KnownAngle(t *testing.T) {
	// 45 degree angle: cos(45°) ≈ 0.7071
	a := []float64{1, 0}
	b := []float64{1, 1} // 45 degrees
	score := cosineSimilarity(a, b)
	expected := 1.0 / math.Sqrt(2)
	if math.Abs(score-expected) > 1e-9 {
		t.Errorf("45° angle should have cos ≈ %f, got %f", expected, score)
	}
}

func TestCosineSimilarity_EmptyVectors(t *testing.T) {
	score := cosineSimilarity([]float64{}, []float64{})
	if score != 0 {
		t.Errorf("Empty vectors should return 0, got %f", score)
	}
}

func TestCosineSimilarity_HighDimensional(t *testing.T) {
	// 1536-dimensional vectors (OpenAI embedding size)
	n := 1536
	a := make([]float64, n)
	b := make([]float64, n)
	for i := 0; i < n; i++ {
		a[i] = float64(i) * 0.001
		b[i] = float64(i) * 0.001
	}
	score := cosineSimilarity(a, b)
	if math.Abs(score-1.0) > 1e-6 {
		t.Errorf("Identical 1536-dim vectors should have similarity ≈1.0, got %f", score)
	}
}

// ===== parseEmbedding =====

func TestParseEmbedding_Valid(t *testing.T) {
	s := `[1.0, 2.0, 3.0]`
	vec, err := parseEmbedding(s)
	if err != nil {
		t.Fatalf("parseEmbedding error: %v", err)
	}
	if len(vec) != 3 {
		t.Fatalf("Expected length 3, got %d", len(vec))
	}
	if vec[0] != 1.0 || vec[1] != 2.0 || vec[2] != 3.0 {
		t.Errorf("Expected [1,2,3], got %v", vec)
	}
}

func TestParseEmbedding_Empty(t *testing.T) {
	vec, err := parseEmbedding(`[]`)
	if err != nil {
		t.Fatalf("parseEmbedding([]) error: %v", err)
	}
	if len(vec) != 0 {
		t.Errorf("Expected empty slice, got %v", vec)
	}
}

func TestParseEmbedding_InvalidJSON(t *testing.T) {
	_, err := parseEmbedding("not json at all")
	if err == nil {
		t.Error("parseEmbedding on invalid JSON should return error")
	}
}

func TestParseEmbedding_NestedArray(t *testing.T) {
	// Should fail — we expect flat arrays, not nested
	_, err := parseEmbedding(`[[1,2],[3,4]]`)
	if err == nil {
		t.Error("parseEmbedding on nested array should return error")
	}
}

func TestParseEmbedding_LargeVector(t *testing.T) {
	// Simulate a 1536-dim embedding as JSON
	parts := make([]byte, 0, 20000)
	parts = append(parts, '[')
	for i := 0; i < 1536; i++ {
		if i > 0 {
			parts = append(parts, ',')
		}
		parts = append(parts, []byte("0.123456")...)
	}
	parts = append(parts, ']')

	vec, err := parseEmbedding(string(parts))
	if err != nil {
		t.Fatalf("parseEmbedding(large) error: %v", err)
	}
	if len(vec) != 1536 {
		t.Errorf("Expected 1536 dims, got %d", len(vec))
	}
}
