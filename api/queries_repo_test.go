package api

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGitHubRepo_notFound(t *testing.T) {
	httpReg := &httpmock.Registry{}
	defer httpReg.Verify(t)

	httpReg.Register(
		httpmock.GraphQL(`query RepositoryInfo\b`),
		httpmock.StringResponse(`{ "data": { "repository": null } }`))

	client := newTestClient(httpReg)
	repo, err := GitHubRepo(client, ghrepo.New("OWNER", "REPO"))
	if err == nil {
		t.Fatal("GitHubRepo did not return an error")
	}
	if wants := "GraphQL: Could not resolve to a Repository with the name 'OWNER/REPO'."; err.Error() != wants {
		t.Errorf("GitHubRepo error: want %q, got %q", wants, err.Error())
	}
	if repo != nil {
		t.Errorf("GitHubRepo: expected nil repo, got %v", repo)
	}
}

func Test_RepoMetadata(t *testing.T) {
	http := &httpmock.Registry{}
	client := newTestClient(http)

	repo, _ := ghrepo.FromFullName("OWNER/REPO")
	input := RepoMetadataInput{
		Assignees:  true,
		Reviewers:  true,
		Labels:     true,
		Projects:   true,
		Milestones: true,
	}

	http.Register(
		httpmock.GraphQL(`query RepositoryAssignableUsers\b`),
		httpmock.StringResponse(`
		{ "data": { "repository": { "assignableUsers": {
			"nodes": [
				{ "login": "hubot", "id": "HUBOTID" },
				{ "login": "MonaLisa", "id": "MONAID" }
			],
			"pageInfo": { "hasNextPage": false }
		} } } }
		`))
	http.Register(
		httpmock.GraphQL(`query RepositoryLabelList\b`),
		httpmock.StringResponse(`
		{ "data": { "repository": { "labels": {
			"nodes": [
				{ "name": "feature", "id": "FEATUREID" },
				{ "name": "TODO", "id": "TODOID" },
				{ "name": "bug", "id": "BUGID" }
			],
			"pageInfo": { "hasNextPage": false }
		} } } }
		`))
	http.Register(
		httpmock.GraphQL(`query RepositoryMilestoneList\b`),
		httpmock.StringResponse(`
		{ "data": { "repository": { "milestones": {
			"nodes": [
				{ "title": "GA", "id": "GAID" },
				{ "title": "Big One.oh", "id": "BIGONEID" }
			],
			"pageInfo": { "hasNextPage": false }
		} } } }
		`))
	http.Register(
		httpmock.GraphQL(`query RepositoryProjectList\b`),
		httpmock.StringResponse(`
		{ "data": { "repository": { "projects": {
			"nodes": [
				{ "name": "Cleanup", "id": "CLEANUPID" },
				{ "name": "Roadmap", "id": "ROADMAPID" }
			],
			"pageInfo": { "hasNextPage": false }
		} } } }
		`))
	http.Register(
		httpmock.GraphQL(`query RepositoryProjectV2List\b`),
		httpmock.StringResponse(`
		{ "data": { "repository": { "projectsV2": {
			"nodes": [
				{ "title": "CleanupV2", "id": "CLEANUPV2ID" },
				{ "title": "RoadmapV2", "id": "ROADMAPV2ID" }
			],
			"pageInfo": { "hasNextPage": false }
		} } } }
		`))
	http.Register(
		httpmock.GraphQL(`query OrganizationProjectList\b`),
		httpmock.StringResponse(`
		{ "data": { "organization": { "projects": {
			"nodes": [
				{ "name": "Triage", "id": "TRIAGEID" }
			],
			"pageInfo": { "hasNextPage": false }
		} } } }
		`))
	http.Register(
		httpmock.GraphQL(`query OrganizationProjectV2List\b`),
		httpmock.StringResponse(`
		{ "data": { "organization": { "projectsV2": {
			"nodes": [
				{ "title": "TriageV2", "id": "TRIAGEV2ID" }
			],
			"pageInfo": { "hasNextPage": false }
		} } } }
		`))
	http.Register(
		httpmock.GraphQL(`query UserProjectV2List\b`),
		httpmock.StringResponse(`
		{ "data": { "viewer": { "projectsV2": {
			"nodes": [
				{ "title": "MonalisaV2", "id": "MONALISAV2ID" }
			],
			"pageInfo": { "hasNextPage": false }
		} } } }
		`))
	http.Register(
		httpmock.GraphQL(`query OrganizationTeamList\b`),
		httpmock.StringResponse(`
		{ "data": { "organization": { "teams": {
			"nodes": [
				{ "slug": "owners", "id": "OWNERSID" },
				{ "slug": "Core", "id": "COREID" }
			],
			"pageInfo": { "hasNextPage": false }
		} } } }
		`))
	http.Register(
		httpmock.GraphQL(`query UserCurrent\b`),
		httpmock.StringResponse(`
		  { "data": { "viewer": { "login": "monalisa" } } }
		`))

	result, err := RepoMetadata(client, repo, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedMemberIDs := []string{"MONAID", "HUBOTID"}
	memberIDs, err := result.MembersToIDs([]string{"monalisa", "hubot"})
	if err != nil {
		t.Errorf("error resolving members: %v", err)
	}
	if !sliceEqual(memberIDs, expectedMemberIDs) {
		t.Errorf("expected members %v, got %v", expectedMemberIDs, memberIDs)
	}

	expectedTeamIDs := []string{"COREID", "OWNERSID"}
	teamIDs, err := result.TeamsToIDs([]string{"OWNER/core", "/owners"})
	if err != nil {
		t.Errorf("error resolving teams: %v", err)
	}
	if !sliceEqual(teamIDs, expectedTeamIDs) {
		t.Errorf("expected teams %v, got %v", expectedTeamIDs, teamIDs)
	}

	expectedLabelIDs := []string{"BUGID", "TODOID"}
	labelIDs, err := result.LabelsToIDs([]string{"bug", "todo"})
	if err != nil {
		t.Errorf("error resolving labels: %v", err)
	}
	if !sliceEqual(labelIDs, expectedLabelIDs) {
		t.Errorf("expected labels %v, got %v", expectedLabelIDs, labelIDs)
	}

	expectedProjectIDs := []string{"TRIAGEID", "ROADMAPID"}
	expectedProjectV2IDs := []string{"TRIAGEV2ID", "ROADMAPV2ID", "MONALISAV2ID"}
	projectIDs, projectV2IDs, err := result.ProjectsToIDs([]string{"triage", "roadmap", "triagev2", "roadmapv2", "monalisav2"})
	if err != nil {
		t.Errorf("error resolving projects: %v", err)
	}
	if !sliceEqual(projectIDs, expectedProjectIDs) {
		t.Errorf("expected projects %v, got %v", expectedProjectIDs, projectIDs)
	}
	if !sliceEqual(projectV2IDs, expectedProjectV2IDs) {
		t.Errorf("expected projectsV2 %v, got %v", expectedProjectV2IDs, projectV2IDs)
	}

	expectedMilestoneID := "BIGONEID"
	milestoneID, err := result.MilestoneToID("big one.oh")
	if err != nil {
		t.Errorf("error resolving milestone: %v", err)
	}
	if milestoneID != expectedMilestoneID {
		t.Errorf("expected milestone %v, got %v", expectedMilestoneID, milestoneID)
	}

	expectedCurrentLogin := "monalisa"
	if result.CurrentLogin != expectedCurrentLogin {
		t.Errorf("expected current user %v, got %v", expectedCurrentLogin, result.CurrentLogin)
	}
}

func Test_ProjectsToPaths(t *testing.T) {
	expectedProjectPaths := []string{"OWNER/REPO/PROJECT_NUMBER", "ORG/PROJECT_NUMBER", "OWNER/REPO/PROJECT_NUMBER_2"}
	projects := []RepoProject{
		{ID: "id1", Name: "My Project", ResourcePath: "/OWNER/REPO/projects/PROJECT_NUMBER"},
		{ID: "id2", Name: "Org Project", ResourcePath: "/orgs/ORG/projects/PROJECT_NUMBER"},
		{ID: "id3", Name: "Project", ResourcePath: "/orgs/ORG/projects/PROJECT_NUMBER_2"},
	}
	projectsV2 := []ProjectV2{
		{ID: "id4", Title: "My Project V2", ResourcePath: "/OWNER/REPO/projects/PROJECT_NUMBER_2"},
		{ID: "id5", Title: "Org Project V2", ResourcePath: "/orgs/ORG/projects/PROJECT_NUMBER_3"},
	}
	projectNames := []string{"My Project", "Org Project", "My Project V2"}

	projectPaths, err := ProjectsToPaths(projects, projectsV2, projectNames)
	if err != nil {
		t.Errorf("error resolving projects: %v", err)
	}
	if !sliceEqual(projectPaths, expectedProjectPaths) {
		t.Errorf("expected projects %v, got %v", expectedProjectPaths, projectPaths)
	}
}

func Test_ProjectNamesToPaths(t *testing.T) {
	http := &httpmock.Registry{}
	client := newTestClient(http)

	repo, _ := ghrepo.FromFullName("OWNER/REPO")

	http.Register(
		httpmock.GraphQL(`query RepositoryProjectList\b`),
		httpmock.StringResponse(`
		{ "data": { "repository": { "projects": {
			"nodes": [
				{ "name": "Cleanup", "id": "CLEANUPID", "resourcePath": "/OWNER/REPO/projects/1" },
				{ "name": "Roadmap", "id": "ROADMAPID", "resourcePath": "/OWNER/REPO/projects/2" }
			],
			"pageInfo": { "hasNextPage": false }
		} } } }
		`))
	http.Register(
		httpmock.GraphQL(`query OrganizationProjectList\b`),
		httpmock.StringResponse(`
		{ "data": { "organization": { "projects": {
			"nodes": [
				{ "name": "Triage", "id": "TRIAGEID", "resourcePath": "/orgs/ORG/projects/1"  }
			],
			"pageInfo": { "hasNextPage": false }
		} } } }
		`))
	http.Register(
		httpmock.GraphQL(`query RepositoryProjectV2List\b`),
		httpmock.StringResponse(`
		{ "data": { "repository": { "projectsV2": {
			"nodes": [
				{ "title": "CleanupV2", "id": "CLEANUPV2ID", "resourcePath": "/OWNER/REPO/projects/3" },
				{ "title": "RoadmapV2", "id": "ROADMAPV2ID", "resourcePath": "/OWNER/REPO/projects/4" }
			],
			"pageInfo": { "hasNextPage": false }
		} } } }
		`))
	http.Register(
		httpmock.GraphQL(`query OrganizationProjectV2List\b`),
		httpmock.StringResponse(`
		{ "data": { "organization": { "projectsV2": {
			"nodes": [
				{ "title": "TriageV2", "id": "TRIAGEV2ID", "resourcePath": "/orgs/ORG/projects/2"  }
			],
			"pageInfo": { "hasNextPage": false }
		} } } }
		`))
	http.Register(
		httpmock.GraphQL(`query UserProjectV2List\b`),
		httpmock.StringResponse(`
		{ "data": { "viewer": { "projectsV2": {
			"nodes": [
				{ "title": "MonalisaV2", "id": "MONALISAV2ID", "resourcePath": "/users/MONALISA/projects/5"  }
			],
			"pageInfo": { "hasNextPage": false }
		} } } }
		`))

	projectPaths, err := ProjectNamesToPaths(client, repo, []string{"Triage", "Roadmap", "TriageV2", "RoadmapV2", "MonalisaV2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedProjectPaths := []string{"ORG/1", "OWNER/REPO/2", "ORG/2", "OWNER/REPO/4", "MONALISA/5"}
	if !sliceEqual(projectPaths, expectedProjectPaths) {
		t.Errorf("expected projects paths %v, got %v", expectedProjectPaths, projectPaths)
	}
}

func Test_RepoResolveMetadataIDs(t *testing.T) {
	http := &httpmock.Registry{}
	client := newTestClient(http)

	repo, _ := ghrepo.FromFullName("OWNER/REPO")
	input := RepoResolveInput{
		Assignees: []string{"monalisa", "hubot"},
		Reviewers: []string{"monalisa", "octocat", "OWNER/core", "/robots"},
		Labels:    []string{"bug", "help wanted"},
	}

	expectedQuery := `query RepositoryResolveMetadataIDs {
u000: user(login:"monalisa"){id,login}
u001: user(login:"hubot"){id,login}
u002: user(login:"octocat"){id,login}
repository(owner:"OWNER",name:"REPO"){
l000: label(name:"bug"){id,name}
l001: label(name:"help wanted"){id,name}
}
organization(login:"OWNER"){
t000: team(slug:"core"){id,slug}
t001: team(slug:"robots"){id,slug}
}
}
`
	responseJSON := `
	{ "data": {
		"u000": { "login": "MonaLisa", "id": "MONAID" },
		"u001": { "login": "hubot", "id": "HUBOTID" },
		"u002": { "login": "octocat", "id": "OCTOID" },
		"repository": {
			"l000": { "name": "bug", "id": "BUGID" },
			"l001": { "name": "Help Wanted", "id": "HELPID" }
		},
		"organization": {
			"t000": { "slug": "core", "id": "COREID" },
			"t001": { "slug": "Robots", "id": "ROBOTID" }
		}
	} }
	`

	http.Register(
		httpmock.GraphQL(`query RepositoryResolveMetadataIDs\b`),
		httpmock.GraphQLQuery(responseJSON, func(q string, _ map[string]interface{}) {
			if q != expectedQuery {
				t.Errorf("expected query %q, got %q", expectedQuery, q)
			}
		}))

	result, err := RepoResolveMetadataIDs(client, repo, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedMemberIDs := []string{"MONAID", "HUBOTID", "OCTOID"}
	memberIDs, err := result.MembersToIDs([]string{"monalisa", "hubot", "octocat"})
	if err != nil {
		t.Errorf("error resolving members: %v", err)
	}
	if !sliceEqual(memberIDs, expectedMemberIDs) {
		t.Errorf("expected members %v, got %v", expectedMemberIDs, memberIDs)
	}

	expectedTeamIDs := []string{"COREID", "ROBOTID"}
	teamIDs, err := result.TeamsToIDs([]string{"/core", "/robots"})
	if err != nil {
		t.Errorf("error resolving teams: %v", err)
	}
	if !sliceEqual(teamIDs, expectedTeamIDs) {
		t.Errorf("expected members %v, got %v", expectedTeamIDs, teamIDs)
	}

	expectedLabelIDs := []string{"BUGID", "HELPID"}
	labelIDs, err := result.LabelsToIDs([]string{"bug", "help wanted"})
	if err != nil {
		t.Errorf("error resolving labels: %v", err)
	}
	if !sliceEqual(labelIDs, expectedLabelIDs) {
		t.Errorf("expected members %v, got %v", expectedLabelIDs, labelIDs)
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func Test_RepoMilestones(t *testing.T) {
	tests := []struct {
		state   string
		want    string
		wantErr bool
	}{
		{
			state: "open",
			want:  `"states":["OPEN"]`,
		},
		{
			state: "closed",
			want:  `"states":["CLOSED"]`,
		},
		{
			state: "all",
			want:  `"states":["OPEN","CLOSED"]`,
		},
		{
			state:   "invalid state",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		var query string
		reg := &httpmock.Registry{}
		reg.Register(httpmock.MatchAny, func(req *http.Request) (*http.Response, error) {
			buf := new(strings.Builder)
			_, err := io.Copy(buf, req.Body)
			if err != nil {
				return nil, err
			}
			query = buf.String()
			return httpmock.StringResponse("{}")(req)
		})
		client := newTestClient(reg)

		_, err := RepoMilestones(client, ghrepo.New("OWNER", "REPO"), tt.state)
		if (err != nil) != tt.wantErr {
			t.Errorf("RepoMilestones() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if !strings.Contains(query, tt.want) {
			t.Errorf("query does not contain %v", tt.want)
		}
	}
}

func TestDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		assignee RepoAssignee
		want     string
	}{
		{
			name:     "assignee with name",
			assignee: RepoAssignee{"123", "octocat123", "Octavious Cath"},
			want:     "octocat123 (Octavious Cath)",
		},
		{
			name:     "assignee without name",
			assignee: RepoAssignee{"123", "octocat123", ""},
			want:     "octocat123",
		},
	}
	for _, tt := range tests {
		actual := tt.assignee.DisplayName()
		if actual != tt.want {
			t.Errorf("display name was %s wanted %s", actual, tt.want)
		}
	}
}

func TestRepoExists(t *testing.T) {
	tests := []struct {
		name       string
		httpStub   func(*httpmock.Registry)
		repo       ghrepo.Interface
		existCheck bool
		wantErrMsg string
	}{
		{
			name: "repo exists",
			httpStub: func(r *httpmock.Registry) {
				r.Register(
					httpmock.REST("HEAD", "repos/OWNER/REPO"),
					httpmock.StringResponse("{}"),
				)
			},
			repo:       ghrepo.New("OWNER", "REPO"),
			existCheck: true,
			wantErrMsg: "",
		},
		{
			name: "repo does not exists",
			httpStub: func(r *httpmock.Registry) {
				r.Register(
					httpmock.REST("HEAD", "repos/OWNER/REPO"),
					httpmock.StatusStringResponse(404, "Not Found"),
				)
			},
			repo:       ghrepo.New("OWNER", "REPO"),
			existCheck: false,
			wantErrMsg: "",
		},
		{
			name: "http error",
			httpStub: func(r *httpmock.Registry) {
				r.Register(
					httpmock.REST("HEAD", "repos/OWNER/REPO"),
					httpmock.StatusStringResponse(500, "Internal Server Error"),
				)
			},
			repo:       ghrepo.New("OWNER", "REPO"),
			existCheck: false,
			wantErrMsg: "HTTP 500 (https://api.github.com/repos/OWNER/REPO)",
		},
	}
	for _, tt := range tests {
		reg := &httpmock.Registry{}
		if tt.httpStub != nil {
			tt.httpStub(reg)
		}

		client := newTestClient(reg)

		t.Run(tt.name, func(t *testing.T) {
			exist, err := RepoExists(client, ghrepo.New("OWNER", "REPO"))
			if tt.wantErrMsg != "" {
				assert.Equal(t, tt.wantErrMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}

			if exist != tt.existCheck {
				t.Errorf("RepoExists() returns %v, expected %v", exist, tt.existCheck)
				return
			}
		})
	}
}
