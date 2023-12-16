package view

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/api"
	"github.com/cli/cli/v2/internal/browser"
	"github.com/cli/cli/v2/internal/text"
	"github.com/cli/cli/v2/pkg/cmd/pr/shared"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/cli/cli/v2/pkg/markdown"
	"github.com/spf13/cobra"
)

type ViewOptions struct {
	IO      *iostreams.IOStreams
	Browser browser.Browser

	Finder   shared.PRFinder
	Exporter cmdutil.Exporter

	SelectorArg string
	BrowserMode bool
	Comments    bool

	Now func() time.Time
}

func NewCmdView(f *cmdutil.Factory, runF func(*ViewOptions) error) *cobra.Command {
	opts := &ViewOptions{
		IO:      f.IOStreams,
		Browser: f.Browser,
		Now:     time.Now,
	}

	cmd := &cobra.Command{
		Use:   "view [<number> | <url> | <branch>]",
		Short: "View a pull request",
		Long: heredoc.Docf(`
			Display the title, body, and other information about a pull request.

			Without an argument, the pull request that belongs to the current branch
			is displayed.

			With %[1]s--web%[1]s flag, open the pull request in a web browser instead.
		`, "`"),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Finder = shared.NewFinder(f)

			if repoOverride, _ := cmd.Flags().GetString("repo"); repoOverride != "" && len(args) == 0 {
				return cmdutil.FlagErrorf("argument required when using the --repo flag")
			}

			if len(args) > 0 {
				opts.SelectorArg = args[0]
			}

			if runF != nil {
				return runF(opts)
			}
			return viewRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.BrowserMode, "web", "w", false, "Open a pull request in the browser")
	cmd.Flags().BoolVarP(&opts.Comments, "comments", "c", false, "View pull request comments")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, api.PullRequestFields)

	return cmd
}

var defaultFields = []string{
	"url", "number", "title", "state", "body", "author", "autoMergeRequest",
	"isDraft", "maintainerCanModify", "mergeable", "additions", "deletions", "commitsCount",
	"baseRefName", "headRefName", "headRepositoryOwner", "headRepository", "isCrossRepository",
	"reviewRequests", "reviews", "assignees", "labels", "projectCards", "milestone",
	"comments", "reactionGroups", "createdAt", "statusCheckRollup",
}

func viewRun(opts *ViewOptions) error {
	findOptions := shared.FindOptions{
		Selector: opts.SelectorArg,
		Fields:   defaultFields,
	}
	if opts.BrowserMode {
		findOptions.Fields = []string{"url"}
	} else if opts.Exporter != nil {
		findOptions.Fields = opts.Exporter.Fields()
	}
	pr, _, err := opts.Finder.Find(findOptions)
	if err != nil {
		return err
	}

	connectedToTerminal := opts.IO.IsStdoutTTY()

	if opts.BrowserMode {
		openURL := pr.URL
		if connectedToTerminal {
			fmt.Fprintf(opts.IO.ErrOut, "Opening %s in your browser.\n", text.DisplayURL(openURL))
		}
		return opts.Browser.Browse(openURL)
	}

	opts.IO.DetectTerminalTheme()
	if err := opts.IO.StartPager(); err == nil {
		defer opts.IO.StopPager()
	} else {
		fmt.Fprintf(opts.IO.ErrOut, "failed to start pager: %v\n", err)
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, pr)
	}

	if connectedToTerminal {
		return printHumanPrPreview(opts, pr)
	}

	if opts.Comments {
		fmt.Fprint(opts.IO.Out, shared.RawCommentList(pr.Comments, pr.DisplayableReviews()))
		return nil
	}

	return printRawPrPreview(opts.IO, pr)
}

func printRawPrPreview(io *iostreams.IOStreams, pr *api.PullRequest) error {
	out := io.Out
	cs := io.ColorScheme()

	reviewers := prReviewerList(*pr, cs)
	assignees := prAssigneeList(*pr)
	labels := prLabelList(*pr, cs)
	projects := prProjectList(*pr)

	fmt.Fprintf(out, "title:\t%s\n", pr.Title)
	fmt.Fprintf(out, "state:\t%s\n", prStateWithDraft(pr))
	fmt.Fprintf(out, "author:\t%s\n", pr.Author.Login)
	fmt.Fprintf(out, "labels:\t%s\n", labels)
	fmt.Fprintf(out, "assignees:\t%s\n", assignees)
	fmt.Fprintf(out, "reviewers:\t%s\n", reviewers)
	fmt.Fprintf(out, "projects:\t%s\n", projects)
	var milestoneTitle string
	if pr.Milestone != nil {
		milestoneTitle = pr.Milestone.Title
	}
	fmt.Fprintf(out, "milestone:\t%s\n", milestoneTitle)
	fmt.Fprintf(out, "number:\t%d\n", pr.Number)
	fmt.Fprintf(out, "url:\t%s\n", pr.URL)
	fmt.Fprintf(out, "additions:\t%s\n", cs.Green(strconv.Itoa(pr.Additions)))
	fmt.Fprintf(out, "deletions:\t%s\n", cs.Red(strconv.Itoa(pr.Deletions)))
	var autoMerge string
	if pr.AutoMergeRequest == nil {
		autoMerge = "disabled"
	} else {
		autoMerge = fmt.Sprintf("enabled\t%s\t%s",
			pr.AutoMergeRequest.EnabledBy.Login,
			strings.ToLower(pr.AutoMergeRequest.MergeMethod))
	}
	fmt.Fprintf(out, "auto-merge:\t%s\n", autoMerge)

	fmt.Fprintln(out, "--")
	fmt.Fprintln(out, pr.Body)

	return nil
}

func printHumanPrPreview(opts *ViewOptions, pr *api.PullRequest) error {
	out := opts.IO.Out
	cs := opts.IO.ColorScheme()

	// Header (Title and State)
	fmt.Fprintf(out, "%s #%d\n", cs.Bold(pr.Title), pr.Number)
	fmt.Fprintf(out,
		"%s • %s wants to merge %s into %s from %s • %s\n",
		shared.StateTitleWithColor(cs, *pr),
		pr.Author.Login,
		text.Pluralize(pr.Commits.TotalCount, "commit"),
		pr.BaseRefName,
		pr.HeadRefName,
		text.FuzzyAgo(opts.Now(), pr.CreatedAt),
	)

	// added/removed
	fmt.Fprintf(out,
		"%s %s",
		cs.Green("+"+strconv.Itoa(pr.Additions)),
		cs.Red("-"+strconv.Itoa(pr.Deletions)),
	)

	// checks
	checks := pr.ChecksStatus()
	if summary := shared.PrCheckStatusSummaryWithColor(cs, checks); summary != "" {
		fmt.Fprintf(out, " • %s\n", summary)
	} else {
		fmt.Fprintln(out)
	}

	// Reactions
	if reactions := shared.ReactionGroupList(pr.ReactionGroups); reactions != "" {
		fmt.Fprint(out, reactions)
		fmt.Fprintln(out)
	}

	// Metadata
	if reviewers := prReviewerList(*pr, cs); reviewers != "" {
		fmt.Fprint(out, cs.Bold("Reviewers: "))
		fmt.Fprintln(out, reviewers)
	}
	if assignees := prAssigneeList(*pr); assignees != "" {
		fmt.Fprint(out, cs.Bold("Assignees: "))
		fmt.Fprintln(out, assignees)
	}
	if labels := prLabelList(*pr, cs); labels != "" {
		fmt.Fprint(out, cs.Bold("Labels: "))
		fmt.Fprintln(out, labels)
	}
	if projects := prProjectList(*pr); projects != "" {
		fmt.Fprint(out, cs.Bold("Projects: "))
		fmt.Fprintln(out, projects)
	}
	if pr.Milestone != nil {
		fmt.Fprint(out, cs.Bold("Milestone: "))
		fmt.Fprintln(out, pr.Milestone.Title)
	}

	// Auto-Merge status
	autoMerge := pr.AutoMergeRequest
	if autoMerge != nil {
		var mergeMethod string
		switch autoMerge.MergeMethod {
		case "MERGE":
			mergeMethod = "a merge commit"
		case "REBASE":
			mergeMethod = "rebase and merge"
		case "SQUASH":
			mergeMethod = "squash and merge"
		default:
			mergeMethod = fmt.Sprintf("an unknown merge method (%s)", autoMerge.MergeMethod)
		}
		fmt.Fprintf(out,
			"%s %s by %s, using %s\n",
			cs.Bold("Auto-merge:"),
			cs.Green("enabled"),
			autoMerge.EnabledBy.Login,
			mergeMethod,
		)
	}

	// Body
	var md string
	var err error
	if pr.Body == "" {
		md = fmt.Sprintf("\n  %s\n\n", cs.Gray("No description provided"))
	} else {
		md, err = markdown.Render(pr.Body,
			markdown.WithTheme(opts.IO.TerminalTheme()),
			markdown.WithWrap(opts.IO.TerminalWidth()))
		if err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "\n%s\n", md)

	// Reviews and Comments
	if pr.Comments.TotalCount > 0 || pr.Reviews.TotalCount > 0 {
		preview := !opts.Comments
		comments, err := shared.CommentList(opts.IO, pr.Comments, pr.DisplayableReviews(), preview)
		if err != nil {
			return err
		}
		fmt.Fprint(out, comments)
	}

	// Footer
	fmt.Fprintf(out, cs.Gray("View this pull request on GitHub: %s\n"), pr.URL)

	return nil
}

const (
	requestedReviewState        = "REQUESTED" // This is our own state for review request
	approvedReviewState         = "APPROVED"
	changesRequestedReviewState = "CHANGES_REQUESTED"
	commentedReviewState        = "COMMENTED"
	dismissedReviewState        = "DISMISSED"
	pendingReviewState          = "PENDING"
)

type reviewerState struct {
	Name  string
	State string
}

// formattedReviewerState formats a reviewerState with state color
func formattedReviewerState(cs *iostreams.ColorScheme, reviewer *reviewerState) string {
	var displayState string
	switch reviewer.State {
	case requestedReviewState:
		displayState = cs.Yellow("Requested")
	case approvedReviewState:
		displayState = cs.Green("Approved")
	case changesRequestedReviewState:
		displayState = cs.Red("Changes requested")
	case commentedReviewState, dismissedReviewState:
		// Show "DISMISSED" review as "COMMENTED", since "dismissed" only makes
		// sense when displayed in an events timeline but not in the final tally.
		displayState = "Commented"
	default:
		displayState = text.Title(reviewer.State)
	}

	return fmt.Sprintf("%s (%s)", reviewer.Name, displayState)
}

// prReviewerList generates a reviewer list with their last state
func prReviewerList(pr api.PullRequest, cs *iostreams.ColorScheme) string {
	reviewerStates := parseReviewers(pr)
	reviewers := make([]string, 0, len(reviewerStates))

	sortReviewerStates(reviewerStates)

	for _, reviewer := range reviewerStates {
		reviewers = append(reviewers, formattedReviewerState(cs, reviewer))
	}

	reviewerList := strings.Join(reviewers, ", ")

	return reviewerList
}

const ghostName = "ghost"

// parseReviewers parses given Reviews and ReviewRequests
func parseReviewers(pr api.PullRequest) []*reviewerState {
	reviewerStates := make(map[string]*reviewerState)

	for _, review := range pr.Reviews.Nodes {
		if review.Author.Login != pr.Author.Login {
			name := review.Author.Login
			if name == "" {
				name = ghostName
			}
			reviewerStates[name] = &reviewerState{
				Name:  name,
				State: review.State,
			}
		}
	}

	// Overwrite reviewer's state if a review request for the same reviewer exists.
	for _, reviewRequest := range pr.ReviewRequests.Nodes {
		name := reviewRequest.RequestedReviewer.LoginOrSlug()
		reviewerStates[name] = &reviewerState{
			Name:  name,
			State: requestedReviewState,
		}
	}

	// Convert map to slice for ease of sort
	result := make([]*reviewerState, 0, len(reviewerStates))
	for _, reviewer := range reviewerStates {
		if reviewer.State == pendingReviewState {
			continue
		}
		result = append(result, reviewer)
	}

	return result
}

// sortReviewerStates puts completed reviews before review requests and sorts names alphabetically
func sortReviewerStates(reviewerStates []*reviewerState) {
	sort.Slice(reviewerStates, func(i, j int) bool {
		if reviewerStates[i].State == requestedReviewState &&
			reviewerStates[j].State != requestedReviewState {
			return false
		}
		if reviewerStates[j].State == requestedReviewState &&
			reviewerStates[i].State != requestedReviewState {
			return true
		}

		return reviewerStates[i].Name < reviewerStates[j].Name
	})
}

func prAssigneeList(pr api.PullRequest) string {
	if len(pr.Assignees.Nodes) == 0 {
		return ""
	}

	AssigneeNames := make([]string, 0, len(pr.Assignees.Nodes))
	for _, assignee := range pr.Assignees.Nodes {
		AssigneeNames = append(AssigneeNames, assignee.Login)
	}

	list := strings.Join(AssigneeNames, ", ")
	if pr.Assignees.TotalCount > len(pr.Assignees.Nodes) {
		list += ", …"
	}
	return list
}

func prLabelList(pr api.PullRequest, cs *iostreams.ColorScheme) string {
	if len(pr.Labels.Nodes) == 0 {
		return ""
	}

	// ignore case sort
	sort.SliceStable(pr.Labels.Nodes, func(i, j int) bool {
		return strings.ToLower(pr.Labels.Nodes[i].Name) < strings.ToLower(pr.Labels.Nodes[j].Name)
	})

	labelNames := make([]string, 0, len(pr.Labels.Nodes))
	for _, label := range pr.Labels.Nodes {
		labelNames = append(labelNames, cs.HexToRGB(label.Color, label.Name))
	}

	list := strings.Join(labelNames, ", ")
	if pr.Labels.TotalCount > len(pr.Labels.Nodes) {
		list += ", …"
	}
	return list
}

func prProjectList(pr api.PullRequest) string {
	if len(pr.ProjectCards.Nodes) == 0 {
		return ""
	}

	projectNames := make([]string, 0, len(pr.ProjectCards.Nodes))
	for _, project := range pr.ProjectCards.Nodes {
		colName := project.Column.Name
		if colName == "" {
			colName = "Awaiting triage"
		}
		projectNames = append(projectNames, fmt.Sprintf("%s (%s)", project.Project.Name, colName))
	}

	list := strings.Join(projectNames, ", ")
	if pr.ProjectCards.TotalCount > len(pr.ProjectCards.Nodes) {
		list += ", …"
	}
	return list
}

func prStateWithDraft(pr *api.PullRequest) string {
	if pr.IsDraft && pr.State == "OPEN" {
		return "DRAFT"
	}

	return pr.State
}
