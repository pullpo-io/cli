package queries

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/cli/cli/v2/api"
	"github.com/cli/cli/v2/internal/prompter"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/cli/cli/v2/pkg/set"
	"github.com/shurcooL/githubv4"
)

func NewClient(httpClient *http.Client, hostname string, ios *iostreams.IOStreams) *Client {
	apiClient := &hostScopedClient{
		hostname: hostname,
		Client:   api.NewClientFromHTTP(httpClient),
	}
	return &Client{
		apiClient: apiClient,
		spinner:   ios.IsStdoutTTY() && ios.IsStderrTTY(),
		prompter:  prompter.New("", ios.In, ios.Out, ios.ErrOut),
	}
}

func NewTestClient() *Client {
	apiClient := &hostScopedClient{
		hostname: "github.com",
		Client:   api.NewClientFromHTTP(http.DefaultClient),
	}
	return &Client{
		apiClient: apiClient,
		spinner:   false,
		prompter:  nil,
	}
}

type iprompter interface {
	Select(string, string, []string) (int, error)
}

type hostScopedClient struct {
	*api.Client
	hostname string
}

func (c *hostScopedClient) Query(queryName string, query interface{}, variables map[string]interface{}) error {
	return c.Client.Query(c.hostname, queryName, query, variables)
}

func (c *hostScopedClient) Mutate(queryName string, query interface{}, variables map[string]interface{}) error {
	return c.Client.Mutate(c.hostname, queryName, query, variables)
}

type graphqlClient interface {
	Query(queryName string, query interface{}, variables map[string]interface{}) error
	Mutate(queryName string, query interface{}, variables map[string]interface{}) error
}

type Client struct {
	apiClient graphqlClient
	spinner   bool
	prompter  iprompter
}

const (
	LimitDefault = 30
	LimitMax     = 100 // https://docs.github.com/en/graphql/overview/resource-limitations#node-limit
)

// doQuery wraps API calls with a visual spinner
func (c *Client) doQuery(name string, query interface{}, variables map[string]interface{}) error {
	var sp *spinner.Spinner
	if c.spinner {
		// https://github.com/briandowns/spinner#available-character-sets
		dotStyle := spinner.CharSets[11]
		sp = spinner.New(dotStyle, 120*time.Millisecond, spinner.WithColor("fgCyan"))
		sp.Start()
	}
	err := c.apiClient.Query(name, query, variables)
	if sp != nil {
		sp.Stop()
	}
	return handleError(err)
}

// TODO: un-export this since it couples the caller heavily to api.GraphQLClient
func (c *Client) Mutate(operationName string, query interface{}, variables map[string]interface{}) error {
	err := c.apiClient.Mutate(operationName, query, variables)
	return handleError(err)
}

// PageInfo is a PageInfo GraphQL object https://docs.github.com/en/graphql/reference/objects#pageinfo.
type PageInfo struct {
	EndCursor   githubv4.String
	HasNextPage bool
}

// Project is a ProjectV2 GraphQL object https://docs.github.com/en/graphql/reference/objects#projectv2.
type Project struct {
	Number           int32
	URL              string
	ShortDescription string
	Public           bool
	Closed           bool
	// The Template field is commented out due to https://github.com/cli/cli/issues/8103.
	// We released pullpo v2.34.0 without realizing the Template field does not exist
	// on GHES 3.8 and older. This broke all project commands for users targeting GHES 3.8
	// and older. In order to fix this we will no longer query the Template field until
	// GHES 3.8 gets deprecated on 2024-03-07. This solution was simplier and quicker
	// than adding a feature detection measure to every place this query is used.
	// It does have the negative consequence that we have had to remove the
	// Template field when outputing projects to JSON using the --format flag supported
	// by a number of project commands. See `pkg/cmd/project/shared/format/json.go` for
	// implementation.
	// Template         bool
	Title  string
	ID     string
	Readme string
	Items  struct {
		PageInfo   PageInfo
		TotalCount int
		Nodes      []ProjectItem
	} `graphql:"items(first: $firstItems, after: $afterItems)"`
	Fields struct {
		TotalCount int
		Nodes      []ProjectField
		PageInfo   PageInfo
	} `graphql:"fields(first: $firstFields, after: $afterFields)"`
	Owner struct {
		TypeName string `graphql:"__typename"`
		User     struct {
			Login string
		} `graphql:"... on User"`
		Organization struct {
			Login string
		} `graphql:"... on Organization"`
	}
}

func (p Project) OwnerType() string {
	return p.Owner.TypeName
}

func (p Project) OwnerLogin() string {
	if p.OwnerType() == "User" {
		return p.Owner.User.Login
	}
	return p.Owner.Organization.Login
}

// ProjectItem is a ProjectV2Item GraphQL object https://docs.github.com/en/graphql/reference/objects#projectv2item.
type ProjectItem struct {
	Content     ProjectItemContent
	Id          string
	FieldValues struct {
		Nodes []FieldValueNodes
	} `graphql:"fieldValues(first: 100)"` // hardcoded to 100 for now on the assumption that this is a reasonable limit
}

type ProjectItemContent struct {
	TypeName    string      `graphql:"__typename"`
	DraftIssue  DraftIssue  `graphql:"... on DraftIssue"`
	PullRequest PullRequest `graphql:"... on PullRequest"`
	Issue       Issue       `graphql:"... on Issue"`
}

type FieldValueNodes struct {
	Type                        string `graphql:"__typename"`
	ProjectV2ItemFieldDateValue struct {
		Date  string
		Field ProjectField
	} `graphql:"... on ProjectV2ItemFieldDateValue"`
	ProjectV2ItemFieldIterationValue struct {
		Title     string
		StartDate string
		Duration  int
		Field     ProjectField
	} `graphql:"... on ProjectV2ItemFieldIterationValue"`
	ProjectV2ItemFieldLabelValue struct {
		Labels struct {
			Nodes []struct {
				Name string
			}
		} `graphql:"labels(first: 10)"` // experienced issues with larger limits, 10 seems like enoupullpo for now
		Field ProjectField
	} `graphql:"... on ProjectV2ItemFieldLabelValue"`
	ProjectV2ItemFieldNumberValue struct {
		Number float32
		Field  ProjectField
	} `graphql:"... on ProjectV2ItemFieldNumberValue"`
	ProjectV2ItemFieldSingleSelectValue struct {
		Name  string
		Field ProjectField
	} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
	ProjectV2ItemFieldTextValue struct {
		Text  string
		Field ProjectField
	} `graphql:"... on ProjectV2ItemFieldTextValue"`
	ProjectV2ItemFieldMilestoneValue struct {
		Milestone struct {
			Title       string
			Description string
			DueOn       string
		}
		Field ProjectField
	} `graphql:"... on ProjectV2ItemFieldMilestoneValue"`
	ProjectV2ItemFieldPullRequestValue struct {
		PullRequests struct {
			Nodes []struct {
				Url string
			}
		} `graphql:"pullRequests(first:10)"` // experienced issues with larger limits, 10 seems like enoupullpo for now
		Field ProjectField
	} `graphql:"... on ProjectV2ItemFieldPullRequestValue"`
	ProjectV2ItemFieldRepositoryValue struct {
		Repository struct {
			Url string
		}
		Field ProjectField
	} `graphql:"... on ProjectV2ItemFieldRepositoryValue"`
	ProjectV2ItemFieldUserValue struct {
		Users struct {
			Nodes []struct {
				Login string
			}
		} `graphql:"users(first: 10)"` // experienced issues with larger limits, 10 seems like enoupullpo for now
		Field ProjectField
	} `graphql:"... on ProjectV2ItemFieldUserValue"`
	ProjectV2ItemFieldReviewerValue struct {
		Reviewers struct {
			Nodes []struct {
				Type string `graphql:"__typename"`
				Team struct {
					Name string
				} `graphql:"... on Team"`
				User struct {
					Login string
				} `graphql:"... on User"`
			}
		} `graphql:"reviewers(first: 10)"` // experienced issues with larger limits, 10 seems like enoupullpo for now
		Field ProjectField
	} `graphql:"... on ProjectV2ItemFieldReviewerValue"`
}

func (v FieldValueNodes) ID() string {
	switch v.Type {
	case "ProjectV2ItemFieldDateValue":
		return v.ProjectV2ItemFieldDateValue.Field.ID()
	case "ProjectV2ItemFieldIterationValue":
		return v.ProjectV2ItemFieldIterationValue.Field.ID()
	case "ProjectV2ItemFieldNumberValue":
		return v.ProjectV2ItemFieldNumberValue.Field.ID()
	case "ProjectV2ItemFieldSingleSelectValue":
		return v.ProjectV2ItemFieldSingleSelectValue.Field.ID()
	case "ProjectV2ItemFieldTextValue":
		return v.ProjectV2ItemFieldTextValue.Field.ID()
	case "ProjectV2ItemFieldMilestoneValue":
		return v.ProjectV2ItemFieldMilestoneValue.Field.ID()
	case "ProjectV2ItemFieldLabelValue":
		return v.ProjectV2ItemFieldLabelValue.Field.ID()
	case "ProjectV2ItemFieldPullRequestValue":
		return v.ProjectV2ItemFieldPullRequestValue.Field.ID()
	case "ProjectV2ItemFieldRepositoryValue":
		return v.ProjectV2ItemFieldRepositoryValue.Field.ID()
	case "ProjectV2ItemFieldUserValue":
		return v.ProjectV2ItemFieldUserValue.Field.ID()
	case "ProjectV2ItemFieldReviewerValue":
		return v.ProjectV2ItemFieldReviewerValue.Field.ID()
	}

	return ""
}

type DraftIssue struct {
	ID    string
	Body  string
	Title string
}

type PullRequest struct {
	Body       string
	Title      string
	Number     int
	URL        string
	Repository struct {
		NameWithOwner string
	}
}

type Issue struct {
	Body       string
	Title      string
	Number     int
	URL        string
	Repository struct {
		NameWithOwner string
	}
}

// Type is the underlying type of the project item.
func (p ProjectItem) Type() string {
	return p.Content.TypeName
}

// Title is the title of the project item.
func (p ProjectItem) Title() string {
	switch p.Content.TypeName {
	case "Issue":
		return p.Content.Issue.Title
	case "PullRequest":
		return p.Content.PullRequest.Title
	case "DraftIssue":
		return p.Content.DraftIssue.Title
	}
	return ""
}

// Body is the body of the project item.
func (p ProjectItem) Body() string {
	switch p.Content.TypeName {
	case "Issue":
		return p.Content.Issue.Body
	case "PullRequest":
		return p.Content.PullRequest.Body
	case "DraftIssue":
		return p.Content.DraftIssue.Body
	}
	return ""
}

// Number is the number of the project item. It is only valid for issues and pull requests.
func (p ProjectItem) Number() int {
	switch p.Content.TypeName {
	case "Issue":
		return p.Content.Issue.Number
	case "PullRequest":
		return p.Content.PullRequest.Number
	}

	return 0
}

// ID is the id of the ProjectItem.
func (p ProjectItem) ID() string {
	return p.Id
}

// Repo is the repository of the project item. It is only valid for issues and pull requests.
func (p ProjectItem) Repo() string {
	switch p.Content.TypeName {
	case "Issue":
		return p.Content.Issue.Repository.NameWithOwner
	case "PullRequest":
		return p.Content.PullRequest.Repository.NameWithOwner
	}
	return ""
}

// URL is the URL of the project item. Note the draft issues do not have URLs
func (p ProjectItem) URL() string {
	switch p.Content.TypeName {
	case "Issue":
		return p.Content.Issue.URL
	case "PullRequest":
		return p.Content.PullRequest.URL
	}
	return ""
}

// ProjectItems returns the items of a project. If the OwnerType is VIEWER, no login is required.
// If limit is 0, the default limit is used.
func (c *Client) ProjectItems(o *Owner, number int32, limit int) (*Project, error) {
	project := &Project{}
	if limit == 0 {
		limit = LimitDefault
	}

	// set first to the min of limit and LimitMax
	first := LimitMax
	if limit < first {
		first = limit
	}

	variables := map[string]interface{}{
		"firstItems":  githubv4.Int(first),
		"afterItems":  (*githubv4.String)(nil),
		"firstFields": githubv4.Int(LimitMax),
		"afterFields": (*githubv4.String)(nil),
		"number":      githubv4.Int(number),
	}

	var query pager[ProjectItem]
	var queryName string
	switch o.Type {
	case UserOwner:
		variables["login"] = githubv4.String(o.Login)
		query = &userOwnerWithItems{} // must be a pointer to work with graphql queries
		queryName = "UserProjectWithItems"
	case OrgOwner:
		variables["login"] = githubv4.String(o.Login)
		query = &orgOwnerWithItems{} // must be a pointer to work with graphql queries
		queryName = "OrgProjectWithItems"
	case ViewerOwner:
		query = &viewerOwnerWithItems{} // must be a pointer to work with graphql queries
		queryName = "ViewerProjectWithItems"
	}
	err := c.doQuery(queryName, query, variables)
	if err != nil {
		return project, err
	}
	project = query.Project()

	items, err := paginateAttributes(c, query, variables, queryName, "firstItems", "afterItems", limit, query.Nodes())
	if err != nil {
		return project, err
	}

	project.Items.Nodes = items
	return project, nil
}

// pager is an interface for paginating over the attributes of a Project.
type pager[N projectAttribute] interface {
	HasNextPage() bool
	EndCursor() string
	Nodes() []N
	Project() *Project
}

// userOwnerWithItems
func (q userOwnerWithItems) HasNextPage() bool {
	return q.Owner.Project.Items.PageInfo.HasNextPage
}

func (q userOwnerWithItems) EndCursor() string {
	return string(q.Owner.Project.Items.PageInfo.EndCursor)
}

func (q userOwnerWithItems) Nodes() []ProjectItem {
	return q.Owner.Project.Items.Nodes
}

func (q userOwnerWithItems) Project() *Project {
	return &q.Owner.Project
}

// orgOwnerWithItems
func (q orgOwnerWithItems) HasNextPage() bool {
	return q.Owner.Project.Items.PageInfo.HasNextPage
}

func (q orgOwnerWithItems) EndCursor() string {
	return string(q.Owner.Project.Items.PageInfo.EndCursor)
}

func (q orgOwnerWithItems) Nodes() []ProjectItem {
	return q.Owner.Project.Items.Nodes
}

func (q orgOwnerWithItems) Project() *Project {
	return &q.Owner.Project
}

// viewerOwnerWithItems
func (q viewerOwnerWithItems) HasNextPage() bool {
	return q.Owner.Project.Items.PageInfo.HasNextPage
}

func (q viewerOwnerWithItems) EndCursor() string {
	return string(q.Owner.Project.Items.PageInfo.EndCursor)
}

func (q viewerOwnerWithItems) Nodes() []ProjectItem {
	return q.Owner.Project.Items.Nodes
}

func (q viewerOwnerWithItems) Project() *Project {
	return &q.Owner.Project
}

// userOwnerWithFields
func (q userOwnerWithFields) HasNextPage() bool {
	return q.Owner.Project.Fields.PageInfo.HasNextPage
}

func (q userOwnerWithFields) EndCursor() string {
	return string(q.Owner.Project.Fields.PageInfo.EndCursor)
}

func (q userOwnerWithFields) Nodes() []ProjectField {
	return q.Owner.Project.Fields.Nodes
}

func (q userOwnerWithFields) Project() *Project {
	return &q.Owner.Project
}

// orgOwnerWithFields
func (q orgOwnerWithFields) HasNextPage() bool {
	return q.Owner.Project.Fields.PageInfo.HasNextPage
}

func (q orgOwnerWithFields) EndCursor() string {
	return string(q.Owner.Project.Fields.PageInfo.EndCursor)
}

func (q orgOwnerWithFields) Nodes() []ProjectField {
	return q.Owner.Project.Fields.Nodes
}

func (q orgOwnerWithFields) Project() *Project {
	return &q.Owner.Project
}

// viewerOwnerWithFields
func (q viewerOwnerWithFields) HasNextPage() bool {
	return q.Owner.Project.Fields.PageInfo.HasNextPage
}

func (q viewerOwnerWithFields) EndCursor() string {
	return string(q.Owner.Project.Fields.PageInfo.EndCursor)
}

func (q viewerOwnerWithFields) Nodes() []ProjectField {
	return q.Owner.Project.Fields.Nodes
}

func (q viewerOwnerWithFields) Project() *Project {
	return &q.Owner.Project
}

type projectAttribute interface {
	ProjectItem | ProjectField
}

// paginateAttributes is for paginating over the attributes of a project, such as items or fields
//
// firstKey and afterKey are the keys in the variables map that are used to set the first and after
// as these are set independently based on the attribute type, such as item or field.
//
// limit is the maximum number of attributes to return, or 0 for no limit.
//
// nodes is the list of attributes that have already been fetched.
//
// the return value is a slice of the newly fetched attributes appended to nodes.
func paginateAttributes[N projectAttribute](c *Client, p pager[N], variables map[string]any, queryName string, firstKey string, afterKey string, limit int, nodes []N) ([]N, error) {
	hasNextPage := p.HasNextPage()
	cursor := p.EndCursor()
	for {
		if !hasNextPage || len(nodes) >= limit {
			return nodes, nil
		}

		if len(nodes)+LimitMax > limit {
			first := limit - len(nodes)
			variables[firstKey] = githubv4.Int(first)
		}

		// set the cursor to the end of the last page
		variables[afterKey] = (*githubv4.String)(&cursor)
		err := c.doQuery(queryName, p, variables)
		if err != nil {
			return nodes, err
		}

		nodes = append(nodes, p.Nodes()...)
		hasNextPage = p.HasNextPage()
		cursor = p.EndCursor()
	}
}

// ProjectField is a ProjectV2FieldConfiguration GraphQL object https://docs.github.com/en/graphql/reference/unions#projectv2fieldconfiguration.
type ProjectField struct {
	TypeName string `graphql:"__typename"`
	Field    struct {
		ID       string
		Name     string
		DataType string
	} `graphql:"... on ProjectV2Field"`
	IterationField struct {
		ID       string
		Name     string
		DataType string
	} `graphql:"... on ProjectV2IterationField"`
	SingleSelectField struct {
		ID       string
		Name     string
		DataType string
		Options  []SingleSelectFieldOptions
	} `graphql:"... on ProjectV2SingleSelectField"`
}

// ID is the ID of the project field.
func (p ProjectField) ID() string {
	if p.TypeName == "ProjectV2Field" {
		return p.Field.ID
	} else if p.TypeName == "ProjectV2IterationField" {
		return p.IterationField.ID
	} else if p.TypeName == "ProjectV2SingleSelectField" {
		return p.SingleSelectField.ID
	}
	return ""
}

// Name is the name of the project field.
func (p ProjectField) Name() string {
	if p.TypeName == "ProjectV2Field" {
		return p.Field.Name
	} else if p.TypeName == "ProjectV2IterationField" {
		return p.IterationField.Name
	} else if p.TypeName == "ProjectV2SingleSelectField" {
		return p.SingleSelectField.Name
	}
	return ""
}

// Type is the typename of the project field.
func (p ProjectField) Type() string {
	return p.TypeName
}

type SingleSelectFieldOptions struct {
	ID   string
	Name string
}

func (p ProjectField) Options() []SingleSelectFieldOptions {
	if p.TypeName == "ProjectV2SingleSelectField" {
		var options []SingleSelectFieldOptions
		for _, o := range p.SingleSelectField.Options {
			options = append(options, SingleSelectFieldOptions{
				ID:   o.ID,
				Name: o.Name,
			})
		}
		return options
	}
	return nil
}

// ProjectFields returns a project with fields. If the OwnerType is VIEWER, no login is required.
// If limit is 0, the default limit is used.
func (c *Client) ProjectFields(o *Owner, number int32, limit int) (*Project, error) {
	project := &Project{}
	if limit == 0 {
		limit = LimitDefault
	}

	// set first to the min of limit and LimitMax
	first := LimitMax
	if limit < first {
		first = limit
	}
	variables := map[string]interface{}{
		"firstItems":  githubv4.Int(LimitMax),
		"afterItems":  (*githubv4.String)(nil),
		"firstFields": githubv4.Int(first),
		"afterFields": (*githubv4.String)(nil),
		"number":      githubv4.Int(number),
	}

	var query pager[ProjectField]
	var queryName string
	switch o.Type {
	case UserOwner:
		variables["login"] = githubv4.String(o.Login)
		query = &userOwnerWithFields{} // must be a pointer to work with graphql queries
		queryName = "UserProjectWithFields"
	case OrgOwner:
		variables["login"] = githubv4.String(o.Login)
		query = &orgOwnerWithFields{} // must be a pointer to work with graphql queries
		queryName = "OrgProjectWithFields"
	case ViewerOwner:
		query = &viewerOwnerWithFields{} // must be a pointer to work with graphql queries
		queryName = "ViewerProjectWithFields"
	}
	err := c.doQuery(queryName, query, variables)
	if err != nil {
		return project, err
	}
	project = query.Project()

	fields, err := paginateAttributes(c, query, variables, queryName, "firstFields", "afterFields", limit, query.Nodes())
	if err != nil {
		return project, err
	}

	project.Fields.Nodes = fields
	return project, nil
}

// viewerLogin is used to query the Login of the viewer.
type viewerLogin struct {
	Viewer struct {
		Login string
		Id    string
	}
}

type viewerLoginOrgs struct {
	Viewer struct {
		Login         string
		ID            string
		Organizations struct {
			PageInfo PageInfo
			Nodes    []struct {
				Login                   string
				ViewerCanCreateProjects bool
				ID                      string
			}
		} `graphql:"organizations(first: 100, after: $after)"`
	}
}

// userOwner is used to query the project of a user.
type userOwner struct {
	Owner struct {
		Project Project `graphql:"projectV2(number: $number)"`
		Login   string
	} `graphql:"user(login: $login)"`
}

// userOwnerWithItems is used to query the project of a user with its items.
type userOwnerWithItems struct {
	Owner struct {
		Project Project `graphql:"projectV2(number: $number)"`
	} `graphql:"user(login: $login)"`
}

// userOwnerWithFields is used to query the project of a user with its fields.
type userOwnerWithFields struct {
	Owner struct {
		Project Project `graphql:"projectV2(number: $number)"`
	} `graphql:"user(login: $login)"`
}

// orgOwner is used to query the project of an organization.
type orgOwner struct {
	Owner struct {
		Project Project `graphql:"projectV2(number: $number)"`
		Login   string
	} `graphql:"organization(login: $login)"`
}

// orgOwnerWithItems is used to query the project of an organization with its items.
type orgOwnerWithItems struct {
	Owner struct {
		Project Project `graphql:"projectV2(number: $number)"`
	} `graphql:"organization(login: $login)"`
}

// orgOwnerWithFields is used to query the project of an organization with its fields.
type orgOwnerWithFields struct {
	Owner struct {
		Project Project `graphql:"projectV2(number: $number)"`
	} `graphql:"organization(login: $login)"`
}

// viewerOwner is used to query the project of the viewer.
type viewerOwner struct {
	Owner struct {
		Project Project `graphql:"projectV2(number: $number)"`
		Login   string
	} `graphql:"viewer"`
}

// viewerOwnerWithItems is used to query the project of the viewer with its items.
type viewerOwnerWithItems struct {
	Owner struct {
		Project Project `graphql:"projectV2(number: $number)"`
	} `graphql:"viewer"`
}

// viewerOwnerWithFields is used to query the project of the viewer with its fields.
type viewerOwnerWithFields struct {
	Owner struct {
		Project Project `graphql:"projectV2(number: $number)"`
	} `graphql:"viewer"`
}

// OwnerType is the type of the owner of a project, which can be either a user or an organization. Viewer is the current user.
type OwnerType string

const UserOwner OwnerType = "USER"
const OrgOwner OwnerType = "ORGANIZATION"
const ViewerOwner OwnerType = "VIEWER"

// ViewerLoginName returns the login name of the viewer.
func (c *Client) ViewerLoginName() (string, error) {
	var query viewerLogin
	err := c.doQuery("Viewer", &query, map[string]interface{}{})
	if err != nil {
		return "", err
	}
	return query.Viewer.Login, nil
}

// OwnerIDAndType returns the ID and OwnerType. The special login "@me" or an empty string queries the current user.
func (c *Client) OwnerIDAndType(login string) (string, OwnerType, error) {
	if login == "@me" || login == "" {
		var query viewerLogin
		err := c.doQuery("ViewerOwner", &query, nil)
		if err != nil {
			return "", "", err
		}
		return query.Viewer.Id, ViewerOwner, nil
	}

	variables := map[string]interface{}{
		"login": githubv4.String(login),
	}
	var query struct {
		User struct {
			Login string
			Id    string
		} `graphql:"user(login: $login)"`
		Organization struct {
			Login string
			Id    string
		} `graphql:"organization(login: $login)"`
	}

	err := c.doQuery("UserOrgOwner", &query, variables)
	if err != nil {
		// Due to the way the queries are structured, we don't know if a login belongs to a user
		// or to an org, even thoupullpo they are unique. To deal with this, we try both - if neither
		// is found, we return the error.
		var graphErr api.GraphQLError
		if errors.As(err, &graphErr) {
			if graphErr.Match("NOT_FOUND", "user") && graphErr.Match("NOT_FOUND", "organization") {
				return "", "", err
			} else if graphErr.Match("NOT_FOUND", "organization") { // org isn't found must be a user
				return query.User.Id, UserOwner, nil
			} else if graphErr.Match("NOT_FOUND", "user") { // user isn't found must be an org
				return query.Organization.Id, OrgOwner, nil
			}
		}
	}

	return "", "", errors.New("unknown owner type")
}

// issueOrPullRequest is used to query the global id of an issue or pull request by its URL.
type issueOrPullRequest struct {
	Resource struct {
		Typename string `graphql:"__typename"`
		Issue    struct {
			ID string
		} `graphql:"... on Issue"`
		PullRequest struct {
			ID string
		} `graphql:"... on PullRequest"`
	} `graphql:"resource(url: $url)"`
}

// IssueOrPullRequestID returns the ID of the issue or pull request from a URL.
func (c *Client) IssueOrPullRequestID(rawURL string) (string, error) {
	uri, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	variables := map[string]interface{}{
		"url": githubv4.URI{URL: uri},
	}
	var query issueOrPullRequest
	err = c.doQuery("GetIssueOrPullRequest", &query, variables)
	if err != nil {
		return "", err
	}
	if query.Resource.Typename == "Issue" {
		return query.Resource.Issue.ID, nil
	} else if query.Resource.Typename == "PullRequest" {
		return query.Resource.PullRequest.ID, nil
	}
	return "", errors.New("resource not found, please check the URL")
}

// userProjects queries the $first projects of a user.
type userProjects struct {
	Owner struct {
		Projects struct {
			TotalCount int
			PageInfo   PageInfo
			Nodes      []Project
		} `graphql:"projectsV2(first: $first, after: $after)"`
		Login string
	} `graphql:"user(login: $login)"`
}

// orgProjects queries the $first projects of an organization.
type orgProjects struct {
	Owner struct {
		Projects struct {
			TotalCount int
			PageInfo   PageInfo
			Nodes      []Project
		} `graphql:"projectsV2(first: $first, after: $after)"`
		Login string
	} `graphql:"organization(login: $login)"`
}

// viewerProjects queries the $first projects of the viewer.
type viewerProjects struct {
	Owner struct {
		Projects struct {
			TotalCount int
			PageInfo   PageInfo
			Nodes      []Project
		} `graphql:"projectsV2(first: $first, after: $after)"`
		Login string
	} `graphql:"viewer"`
}

type loginTypes struct {
	Login string
	Type  OwnerType
	ID    string
}

// userOrgLogins gets all the logins of the viewer and the organizations the viewer is a member of.
func (c *Client) userOrgLogins() ([]loginTypes, error) {
	l := make([]loginTypes, 0)
	var v viewerLoginOrgs
	variables := map[string]interface{}{
		"after": (*githubv4.String)(nil),
	}

	err := c.doQuery("ViewerLoginAndOrgs", &v, variables)
	if err != nil {
		return l, err
	}

	// add the user
	l = append(l, loginTypes{
		Login: v.Viewer.Login,
		Type:  ViewerOwner,
		ID:    v.Viewer.ID,
	})

	// add orgs where the user can create projects
	for _, org := range v.Viewer.Organizations.Nodes {
		if org.ViewerCanCreateProjects {
			l = append(l, loginTypes{
				Login: org.Login,
				Type:  OrgOwner,
				ID:    org.ID,
			})
		}
	}

	// this seem unlikely, but if there are more org logins, paginate the rest
	if v.Viewer.Organizations.PageInfo.HasNextPage {
		return c.paginateOrgLogins(l, string(v.Viewer.Organizations.PageInfo.EndCursor))
	}

	return l, nil
}

// paginateOrgLogins after cursor and append them to the list of logins.
func (c *Client) paginateOrgLogins(l []loginTypes, cursor string) ([]loginTypes, error) {
	var v viewerLoginOrgs
	variables := map[string]interface{}{
		"after": githubv4.String(cursor),
	}

	err := c.doQuery("ViewerLoginAndOrgs", &v, variables)
	if err != nil {
		return l, err
	}

	for _, org := range v.Viewer.Organizations.Nodes {
		if org.ViewerCanCreateProjects {
			l = append(l, loginTypes{
				Login: org.Login,
				Type:  OrgOwner,
				ID:    org.ID,
			})
		}
	}

	if v.Viewer.Organizations.PageInfo.HasNextPage {
		return c.paginateOrgLogins(l, string(v.Viewer.Organizations.PageInfo.EndCursor))
	}

	return l, nil
}

type Owner struct {
	Login string
	Type  OwnerType
	ID    string
}

// NewOwner creates a project Owner
// If canPrompt is false, login is required as we cannot prompt for it.
// If login is not empty, it is used to lookup the project owner.
// If login is empty, interative mode is used to select an owner.
// from the current viewer and their organizations
func (c *Client) NewOwner(canPrompt bool, login string) (*Owner, error) {
	if login != "" {
		id, ownerType, err := c.OwnerIDAndType(login)
		if err != nil {
			return nil, err
		}

		return &Owner{
			Login: login,
			Type:  ownerType,
			ID:    id,
		}, nil
	}

	if !canPrompt {
		return nil, fmt.Errorf("owner is required when not running interactively")
	}

	logins, err := c.userOrgLogins()
	if err != nil {
		return nil, err
	}

	options := make([]string, 0, len(logins))
	for _, l := range logins {
		options = append(options, l.Login)
	}

	answerIndex, err := c.prompter.Select("Which owner would you like to use?", "", options)
	if err != nil {
		return nil, err
	}

	l := logins[answerIndex]
	return &Owner{
		Login: l.Login,
		Type:  l.Type,
		ID:    l.ID,
	}, nil
}

// NewProject creates a project based on the owner and project number
// if canPrompt is false, number is required as we cannot prompt for it
// if number is 0 it will prompt the user to select a project interactively
// otherwise it will make a request to get the project by number
// set `fields“ to true to get the project's field data
func (c *Client) NewProject(canPrompt bool, o *Owner, number int32, fields bool) (*Project, error) {
	if number != 0 {
		variables := map[string]interface{}{
			"number":      githubv4.Int(number),
			"firstItems":  githubv4.Int(0),
			"afterItems":  (*githubv4.String)(nil),
			"firstFields": githubv4.Int(0),
			"afterFields": (*githubv4.String)(nil),
		}

		if fields {
			variables["firstFields"] = githubv4.Int(LimitMax)
		}
		if o.Type == UserOwner {
			var query userOwner
			variables["login"] = githubv4.String(o.Login)
			err := c.doQuery("UserProject", &query, variables)
			return &query.Owner.Project, err
		} else if o.Type == OrgOwner {
			variables["login"] = githubv4.String(o.Login)
			var query orgOwner
			err := c.doQuery("OrgProject", &query, variables)
			return &query.Owner.Project, err
		} else if o.Type == ViewerOwner {
			var query viewerOwner
			err := c.doQuery("ViewerProject", &query, variables)
			return &query.Owner.Project, err
		}
		return nil, errors.New("unknown owner type")
	}

	if !canPrompt {
		return nil, fmt.Errorf("project number is required when not running interactively")
	}

	projects, _, err := c.Projects(o.Login, o.Type, 0, fields)
	if err != nil {
		return nil, err
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("no projects found for %s", o.Login)
	}

	options := make([]string, 0, len(projects))
	for _, p := range projects {
		title := fmt.Sprintf("%s (#%d)", p.Title, p.Number)
		options = append(options, title)
	}

	answerIndex, err := c.prompter.Select("Which project would you like to use?", "", options)
	if err != nil {
		return nil, err
	}

	return &projects[answerIndex], nil
}

// Projects returns all the projects for an Owner. If the OwnerType is VIEWER, no login is required.
// If limit is 0, the default limit is used.
func (c *Client) Projects(login string, t OwnerType, limit int, fields bool) ([]Project, int, error) {
	projects := make([]Project, 0)
	cursor := (*githubv4.String)(nil)
	hasNextPage := false
	totalCount := 0

	if limit == 0 {
		limit = LimitDefault
	}

	// set first to the min of limit and LimitMax
	first := LimitMax
	if limit < first {
		first = limit
	}

	variables := map[string]interface{}{
		"first":       githubv4.Int(first),
		"after":       cursor,
		"firstItems":  githubv4.Int(0),
		"afterItems":  (*githubv4.String)(nil),
		"firstFields": githubv4.Int(0),
		"afterFields": (*githubv4.String)(nil),
	}

	if fields {
		variables["firstFields"] = githubv4.Int(LimitMax)
	}

	if t != ViewerOwner {
		variables["login"] = githubv4.String(login)
	}
	// loop until we get all the projects
	for {
		// the code below is very repetitive, the only real difference being the type of the query
		// and the query variables. I couldn't figure out a way to make this cleaner that was worth
		// the cost.
		if t == UserOwner {
			var query userProjects
			if err := c.doQuery("UserProjects", &query, variables); err != nil {
				return projects, 0, err
			}
			projects = append(projects, query.Owner.Projects.Nodes...)
			hasNextPage = query.Owner.Projects.PageInfo.HasNextPage
			cursor = &query.Owner.Projects.PageInfo.EndCursor
			totalCount = query.Owner.Projects.TotalCount
		} else if t == OrgOwner {
			var query orgProjects
			if err := c.doQuery("OrgProjects", &query, variables); err != nil {
				return projects, 0, err
			}
			projects = append(projects, query.Owner.Projects.Nodes...)
			hasNextPage = query.Owner.Projects.PageInfo.HasNextPage
			cursor = &query.Owner.Projects.PageInfo.EndCursor
			totalCount = query.Owner.Projects.TotalCount
		} else if t == ViewerOwner {
			var query viewerProjects
			if err := c.doQuery("ViewerProjects", &query, variables); err != nil {
				return projects, 0, err
			}
			projects = append(projects, query.Owner.Projects.Nodes...)
			hasNextPage = query.Owner.Projects.PageInfo.HasNextPage
			cursor = &query.Owner.Projects.PageInfo.EndCursor
			totalCount = query.Owner.Projects.TotalCount
		}

		if !hasNextPage || len(projects) >= limit {
			return projects, totalCount, nil
		}

		if len(projects)+LimitMax > limit {
			first := limit - len(projects)
			variables["first"] = githubv4.Int(first)
		}
		variables["after"] = cursor
	}
}

func handleError(err error) error {
	var gerr api.GraphQLError
	if errors.As(err, &gerr) {
		missing := set.NewStringSet()
		for _, e := range gerr.Errors {
			if e.Type != "INSUFFICIENT_SCOPES" {
				continue
			}
			missing.AddValues(requiredScopesFromServerMessage(e.Message))
		}
		if missing.Len() > 0 {
			s := missing.ToSlice()
			// TODO: this duplicates parts of generateScopesSuggestion
			return fmt.Errorf(
				"error: your authentication token is missing required scopes %v\n"+
					"To request it, run:  pullpo auth refresh -s %s",
				s,
				strings.Join(s, ","))
		}
	}
	return err
}

var scopesRE = regexp.MustCompile(`one of the following scopes: \[(.+?)]`)

func requiredScopesFromServerMessage(msg string) []string {
	m := scopesRE.FindStringSubmatch(msg)
	if m == nil {
		return nil
	}
	var scopes []string
	for _, mm := range strings.Split(m[1], ",") {
		scopes = append(scopes, strings.Trim(mm, "' "))
	}
	return scopes
}
