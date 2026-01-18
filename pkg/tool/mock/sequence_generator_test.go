package mock

import "testing"

func TestRangeSequence(t *testing.T) {
	t.Run("sequence_int", func(t *testing.T) {
		seq, err := NewRangeSequence(SequenceColumn{Name: "id", Type: SequenceTypeInt, Digits: 2}, WithWrap(true))
		if err != nil {
			t.Fatal(err)
		}
		nextRange, b := seq.NextRange(10)
		t.Logf("next range: %d~%d  loop:%v", nextRange.Start, nextRange.End, b)
		nextRange, b = seq.NextRange(10)
		t.Logf("next range: %d~%d  loop:%v", nextRange.Start, nextRange.End, b)
		nextRange, b = seq.NextRange(90)
		t.Logf("next range: %d~%d  loop:%v", nextRange.Start, nextRange.End, b)
		nextRange, b = seq.NextRange(10)
		t.Logf("next range: %d~%d  loop:%v", nextRange.Start, nextRange.End, b)
	})

}

func TestRangeSequenceBounded(t *testing.T) {
	seq, err := NewRangeSequence(SequenceColumn{Name: "id", Type: SequenceTypeInt, Digits: 1}, WithWrap(false))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	range1, done := seq.NextRange(5)
	if done {
		t.Fatalf("expected not done")
	}
	if range1.Start != 0 || range1.End != 4 {
		t.Fatalf("unexpected range: %+v", range1)
	}
	range2, done := seq.NextRange(5)
	if !done {
		t.Fatalf("expected done")
	}
	if range2.Start != 5 || range2.End != 9 {
		t.Fatalf("unexpected range: %+v", range2)
	}
}

func TestRangeSequenceWrap(t *testing.T) {
	seq, err := NewRangeSequence(SequenceColumn{Name: "id", Type: SequenceTypeInt, Digits: 1}, WithWrap(true))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	range1, done := seq.NextRange(6)
	if done {
		t.Fatalf("expected not done")
	}
	if range1.Start != 0 || range1.End != 5 {
		t.Fatalf("unexpected range: %+v", range1)
	}
	range2, done := seq.NextRange(6)
	if done {
		t.Fatalf("expected not done")
	}
	if range2.Start != 6 || range2.End != 9 {
		t.Fatalf("unexpected range: %+v", range2)
	}
	range3, done := seq.NextRange(3)
	if done {
		t.Fatalf("expected not done")
	}
	if range3.Start != 0 || range3.End != 2 {
		t.Fatalf("unexpected range: %+v", range3)
	}
}

func TestRowSequenceString(t *testing.T) {
	seq, err := NewRowSequence([]SequenceColumn{{Name: "code", Type: SequenceTypeString, Length: 2}})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	rows, done := seq.NextRows(3)
	if done {
		t.Fatalf("expected not done")
	}
	if rows[0]["code"] != "aa" || rows[1]["code"] != "ab" || rows[2]["code"] != "ac" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestRowSequenceComposite(t *testing.T) {
	seq, err := NewRowSequence([]SequenceColumn{
		{Name: "code", Type: SequenceTypeString, Length: 2},
		{Name: "seq", Type: SequenceTypeInt, Digits: 1},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	rows, done := seq.NextRows(12)
	if done {
		t.Fatalf("expected not done")
	}
	expected := []struct {
		code string
		seq  int64
	}{
		{"aa", 0},
		{"aa", 1},
		{"aa", 2},
		{"aa", 3},
		{"aa", 4},
		{"aa", 5},
		{"aa", 6},
		{"aa", 7},
		{"aa", 8},
		{"aa", 9},
		{"ab", 0},
		{"ab", 1},
	}
	for i, exp := range expected {
		if rows[i]["code"] != exp.code {
			t.Fatalf("row %d code expected %s got %v", i, exp.code, rows[i]["code"])
		}
		if rows[i]["seq"] != exp.seq {
			t.Fatalf("row %d seq expected %d got %v", i, exp.seq, rows[i]["seq"])
		}
	}
}

func TestRowSequenceExhaust(t *testing.T) {
	seq, err := NewRowSequence([]SequenceColumn{{Name: "code", Type: SequenceTypeString, Length: 1}})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	rows, done := seq.NextRows(26)
	if done {
		t.Fatalf("expected not done")
	}
	if len(rows) != 26 {
		t.Fatalf("expected 26 rows, got %d", len(rows))
	}
	rows, done = seq.NextRows(1)
	if !done {
		t.Fatalf("expected done")
	}
	if len(rows) != 0 {
		t.Fatalf("expected no rows, got %d", len(rows))
	}
}
