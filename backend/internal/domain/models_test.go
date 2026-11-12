package domain

import "testing"

func TestExactStatusTransitions(t *testing.T) {
	cases := []struct {
		role     Role
		from, to string
		allowed  bool
	}{
		{Agent, "open", "in_progress", true},
		{Admin, "open", "in_progress", true},
		{Requester, "open", "in_progress", false},
		{Agent, "in_progress", "resolved", true},
		{Requester, "in_progress", "resolved", false},
		{Requester, "resolved", "closed", true},
		{Agent, "resolved", "closed", true},
		{Admin, "resolved", "closed", true},
		{Requester, "resolved", "open", true},
		{Agent, "resolved", "open", false},
		{Requester, "open", "closed", true},
		{Admin, "open", "closed", true},
		{Agent, "open", "closed", false},
		{Admin, "closed", "open", false},
		{Requester, "open", "resolved", false},
	}
	for _, tc := range cases {
		t.Run(string(tc.role)+"/"+tc.from+"->"+tc.to, func(t *testing.T) {
			if got := CanTransition(tc.role, tc.from, tc.to); got != tc.allowed {
				t.Fatalf("CanTransition(%q, %q, %q) = %v, want %v", tc.role, tc.from, tc.to, got, tc.allowed)
			}
		})
	}
}

func TestRequesterCanAccessOnlyOwnedTickets(t *testing.T) {
	ticket := Ticket{RequesterID: 7}
	if !CanAccessTicket(User{ID: 7, Role: Requester}, ticket) {
		t.Fatal("requester should access their ticket")
	}
	if CanAccessTicket(User{ID: 8, Role: Requester}, ticket) {
		t.Fatal("requester should not access another user's ticket")
	}
}

func TestStaffCanAccessAnyTicket(t *testing.T) {
	ticket := Ticket{RequesterID: 7}
	for _, role := range []Role{Agent, Admin} {
		if !CanAccessTicket(User{ID: 99, Role: role}, ticket) {
			t.Errorf("%s should access any ticket", role)
		}
	}
}

func TestInternalCommentVisibility(t *testing.T) {
	if CanViewInternalComments(User{Role: Requester}) {
		t.Fatal("requester must not view internal comments")
	}
	if !CanViewInternalComments(User{Role: Agent}) || !CanViewInternalComments(User{Role: Admin}) {
		t.Fatal("staff should view internal comments")
	}
}

func TestRoleAndEnumValidation(t *testing.T) {
	for _, role := range []Role{Requester, Agent, Admin} {
		if !ValidRole(role) {
			t.Errorf("expected valid role %q", role)
		}
	}
	if ValidRole("owner") || ValidPriority("critical") || ValidStatus("pending") {
		t.Fatal("unexpected enum value accepted")
	}
}
