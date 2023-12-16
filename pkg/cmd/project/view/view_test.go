package view

import (
	"bytes"
	"testing"

	"github.com/cli/cli/v2/pkg/cmd/project/shared/queries"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func TestNewCmdview(t *testing.T) {
	tests := []struct {
		name        string
		cli         string
		wants       viewOpts
		wantsErr    bool
		wantsErrMsg string
	}{
		{
			name:        "not-a-number",
			cli:         "x",
			wantsErr:    true,
			wantsErrMsg: "invalid number: x",
		},
		{
			name: "number",
			cli:  "123",
			wants: viewOpts{
				number: 123,
			},
		},
		{
			name: "owner",
			cli:  "--owner monalisa",
			wants: viewOpts{
				owner: "monalisa",
			},
		},
		{
			name: "web",
			cli:  "--web",
			wants: viewOpts{
				web: true,
			},
		},
		{
			name: "json",
			cli:  "--format json",
			wants: viewOpts{
				format: "json",
			},
		},
	}

	t.Setenv("GH_TOKEN", "auth-token")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			f := &cmdutil.Factory{
				IOStreams: ios,
			}

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts viewOpts
			cmd := NewCmdView(f, func(config viewConfig) error {
				gotOpts = config.opts
				return nil
			})

			cmd.SetArgs(argv)
			_, err = cmd.ExecuteC()
			if tt.wantsErr {
				assert.Error(t, err)
				assert.Equal(t, tt.wantsErrMsg, err.Error())
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, tt.wants.number, gotOpts.number)
			assert.Equal(t, tt.wants.owner, gotOpts.owner)
			assert.Equal(t, tt.wants.format, gotOpts.format)
			assert.Equal(t, tt.wants.web, gotOpts.web)
		})
	}
}

func TestBuildURLViewer(t *testing.T) {
	defer gock.Off()

	gock.New("https://api.github.com").
		Post("/graphql").
		Reply(200).
		JSON(`
			{"data":
				{"viewer":
					{
						"login":"theviewer"
					}
				}
			}
		`)

	client := queries.NewTestClient()

	url, err := buildURL(viewConfig{
		opts: viewOpts{
			number: 1,
			owner:  "@me",
		},
		client: client,
	})
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/users/theviewer/projects/1", url)
}

func TestRunView_User(t *testing.T) {
	defer gock.Off()

	// get user ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query UserOrgOwner.*",
			"variables": map[string]interface{}{
				"login": "monalisa",
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"user": map[string]interface{}{
					"id": "an ID",
				},
			},
			"errors": []interface{}{
				map[string]interface{}{
					"type": "NOT_FOUND",
					"path": []string{"organization"},
				},
			},
		})

	gock.New("https://api.github.com").
		Post("/graphql").
		Reply(200).
		JSON(`
			{"data":
				{"user":
					{
						"login":"monalisa",
						"projectV2": {
							"number": 1,
							"items": {
								"totalCount": 10
							},
							"readme": null,
							"fields": {
								"nodes": [
									{
										"name": "Title"
									}
								]
							}
						}
					}
				}
			}
		`)

	client := queries.NewTestClient()

	ios, _, _, _ := iostreams.Test()
	config := viewConfig{
		opts: viewOpts{
			owner:  "monalisa",
			number: 1,
		},
		io:     ios,
		client: client,
	}

	err := runView(config)
	assert.NoError(t, err)

}

func TestRunView_Viewer(t *testing.T) {
	defer gock.Off()

	// get viewer ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query ViewerOwner.*",
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"viewer": map[string]interface{}{
					"id": "an ID",
				},
			},
		})

	gock.New("https://api.github.com").
		Post("/graphql").
		Reply(200).
		JSON(`
			{"data":
				{"viewer":
					{
						"login":"monalisa",
						"projectV2": {
							"number": 1,
							"items": {
								"totalCount": 10
							},
							"readme": null,
							"fields": {
								"nodes": [
									{
										"name": "Title"
									}
								]
							}
						}
					}
				}
			}
		`)

	client := queries.NewTestClient()

	ios, _, _, _ := iostreams.Test()
	config := viewConfig{
		opts: viewOpts{
			owner:  "@me",
			number: 1,
		},
		io:     ios,
		client: client,
	}

	err := runView(config)
	assert.NoError(t, err)
}

func TestRunView_Org(t *testing.T) {
	defer gock.Off()

	// get org ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query UserOrgOwner.*",
			"variables": map[string]interface{}{
				"login": "github",
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"organization": map[string]interface{}{
					"id": "an ID",
				},
			},
			"errors": []interface{}{
				map[string]interface{}{
					"type": "NOT_FOUND",
					"path": []string{"user"},
				},
			},
		})

	gock.New("https://api.github.com").
		Post("/graphql").
		Reply(200).
		JSON(`
			{"data":
				{"organization":
					{
						"login":"monalisa",
						"projectV2": {
							"number": 1,
							"items": {
								"totalCount": 10
							},
							"readme": null,
							"fields": {
								"nodes": [
									{
										"name": "Title"
									}
								]
							}
						}
					}
				}
			}
		`)

	client := queries.NewTestClient()

	ios, _, _, _ := iostreams.Test()
	config := viewConfig{
		opts: viewOpts{
			owner:  "github",
			number: 1,
		},
		io:     ios,
		client: client,
	}

	err := runView(config)
	assert.NoError(t, err)
}

func TestRunViewWeb_User(t *testing.T) {
	defer gock.Off()
	gock.Observe(gock.DumpRequest)
	// get user ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query UserOrgOwner.*",
			"variables": map[string]interface{}{
				"login": "monalisa",
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"user": map[string]interface{}{
					"id": "an ID",
				},
			},
			"errors": []interface{}{
				map[string]interface{}{
					"type": "NOT_FOUND",
					"path": []string{"organization"},
				},
			},
		})

	gock.New("https://api.github.com").
		Post("/graphql").
		Reply(200).
		JSON(`
		{"data":
			{"user":
				{
					"login":"monalisa",
					"projectV2": {
						"number": 1,
						"items": {
							"totalCount": 10
						},
						"readme": null,
						"fields": {
							"nodes": [
								{
									"name": "Title"
								}
							]
						}
					}
				}
			}
		}
	`)

	client := queries.NewTestClient()
	buf := bytes.Buffer{}
	config := viewConfig{
		opts: viewOpts{
			owner:  "monalisa",
			web:    true,
			number: 8,
		},
		URLOpener: func(url string) error {
			buf.WriteString(url)
			return nil
		},
		client: client,
	}

	err := runView(config)
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/users/monalisa/projects/8", buf.String())
}

func TestRunViewWeb_Org(t *testing.T) {
	defer gock.Off()
	gock.Observe(gock.DumpRequest)
	// get org ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query UserOrgOwner.*",
			"variables": map[string]interface{}{
				"login": "github",
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"organization": map[string]interface{}{
					"id": "an ID",
				},
			},
			"errors": []interface{}{
				map[string]interface{}{
					"type": "NOT_FOUND",
					"path": []string{"user"},
				},
			},
		})

	gock.New("https://api.github.com").
		Post("/graphql").
		Reply(200).
		JSON(`
		{"data":
			{"organization":
				{
					"login":"github",
					"projectV2": {
						"number": 1,
						"items": {
							"totalCount": 10
						},
						"readme": null,
						"fields": {
							"nodes": [
								{
									"name": "Title"
								}
							]
						}
					}
				}
			}
		}
	`)

	client := queries.NewTestClient()
	buf := bytes.Buffer{}
	config := viewConfig{
		opts: viewOpts{
			owner:  "github",
			web:    true,
			number: 8,
		},
		URLOpener: func(url string) error {
			buf.WriteString(url)
			return nil
		},
		client: client,
	}

	err := runView(config)
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/orgs/github/projects/8", buf.String())
}

func TestRunViewWeb_Me(t *testing.T) {
	defer gock.Off()
	gock.Observe(gock.DumpRequest)
	// get viewer ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query Viewer.*",
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"viewer": map[string]interface{}{
					"id":    "an ID",
					"login": "theviewer",
				},
			},
		})

	gock.New("https://api.github.com").
		Post("/graphql").
		Reply(200).
		JSON(`
		{"data":
			{"user":
				{
					"login":"github",
					"projectV2": {
						"number": 1,
						"items": {
							"totalCount": 10
						},
						"readme": null,
						"fields": {
							"nodes": [
								{
									"name": "Title"
								}
							]
						}
					}
				}
			}
		}
	`)

	client := queries.NewTestClient()
	buf := bytes.Buffer{}
	config := viewConfig{
		opts: viewOpts{
			owner:  "@me",
			web:    true,
			number: 8,
		},
		URLOpener: func(url string) error {
			buf.WriteString(url)
			return nil
		},
		client: client,
	}

	err := runView(config)
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/users/theviewer/projects/8", buf.String())
}
