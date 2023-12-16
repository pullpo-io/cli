package api

import "testing"

func TestPullRequestGraphQL(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
		want   string
	}{
		{
			name:   "empty",
			fields: []string(nil),
			want:   "",
		},
		{
			name:   "simple fields",
			fields: []string{"number", "title"},
			want:   "number,title",
		},
		{
			name:   "fields with nested structures",
			fields: []string{"author", "assignees"},
			want:   "author{login,...on User{id,name}},assignees(first:100){nodes{id,login,name},totalCount}",
		},
		{
			name:   "compressed query",
			fields: []string{"files"},
			want:   "files(first: 100) {nodes {additions,deletions,path}}",
		},
		{
			name:   "invalid fields",
			fields: []string{"isPinned", "stateReason", "number"},
			want:   "number",
		},
		{
			name:   "projectItems",
			fields: []string{"projectItems"},
			want:   `projectItems(first:100){nodes{id, project{id,title}, status:fieldValueByName(name: "Status") { ... on ProjectV2ItemFieldSingleSelectValue{optionId,name}}},totalCount}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PullRequestGraphQL(tt.fields); got != tt.want {
				t.Errorf("PullRequestGraphQL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIssueGraphQL(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
		want   string
	}{
		{
			name:   "empty",
			fields: []string(nil),
			want:   "",
		},
		{
			name:   "simple fields",
			fields: []string{"number", "title"},
			want:   "number,title",
		},
		{
			name:   "fields with nested structures",
			fields: []string{"author", "assignees"},
			want:   "author{login,...on User{id,name}},assignees(first:100){nodes{id,login,name},totalCount}",
		},
		{
			name:   "compressed query",
			fields: []string{"files"},
			want:   "files(first: 100) {nodes {additions,deletions,path}}",
		},
		{
			name:   "projectItems",
			fields: []string{"projectItems"},
			want:   `projectItems(first:100){nodes{id, project{id,title}, status:fieldValueByName(name: "Status") { ... on ProjectV2ItemFieldSingleSelectValue{optionId,name}}},totalCount}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IssueGraphQL(tt.fields); got != tt.want {
				t.Errorf("IssueGraphQL() = %v, want %v", got, tt.want)
			}
		})
	}
}
