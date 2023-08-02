package task

import "testing"

func TestContains(t *testing.T) {
	states := []State{Pending, Running, Completed}

	got := Contains(states, Scheduled)
	want := false

	if want != got {
		t.Errorf("want: %t, got: %t", want, got)
	}

	got = Contains(states, Completed)
	want = true

	if want != got {
		t.Errorf("want: %t, got: %t", want, got)
	}

	got = Contains(states, 1)
	want = false

	if want != got {
		t.Errorf("want: %t, got: %t", want, got)
	}
}

func TestValidStateTransition(t *testing.T) {
	got := ValidStateTransition(Pending, Scheduled)
	want := true

	if want != got {
		t.Errorf("want: %t, got: %t", want, got)
	}

	got = ValidStateTransition(Running, Completed)
	want = true

	if want != got {
		t.Errorf("want: %t, got: %t", want, got)
	}

	got = ValidStateTransition(Failed, Completed)
	want = false

	if want != got {
		t.Errorf("want: %t, got: %t", want, got)
	}
}
