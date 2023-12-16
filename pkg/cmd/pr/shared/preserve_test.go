package shared

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/cli/cli/v2/test"
	"github.com/stretchr/testify/assert"
)

func Test_PreserveInput(t *testing.T) {
	tests := []struct {
		name             string
		state            *IssueMetadataState
		err              bool
		wantErrLine      string
		wantPreservation bool
	}{
		{
			name: "err, no changes to state",
			err:  true,
		},
		{
			name: "no err, no changes to state",
			err:  false,
		},
		{
			name: "no err, changes to state",
			state: &IssueMetadataState{
				dirty: true,
			},
		},
		{
			name: "err, title/body input received",
			state: &IssueMetadataState{
				dirty:     true,
				Title:     "almost a",
				Body:      "jill sandwich",
				Reviewers: []string{"barry", "chris"},
				Labels:    []string{"sandwich"},
			},
			wantErrLine:      `X operation failed. To restore: pullpo issue create --recover .*testfile.*`,
			err:              true,
			wantPreservation: true,
		},
		{
			name: "err, metadata received",
			state: &IssueMetadataState{
				Reviewers: []string{"barry", "chris"},
				Labels:    []string{"sandwich"},
			},
			wantErrLine:      `X operation failed. To restore: pullpo issue create --recover .*testfile.*`,
			err:              true,
			wantPreservation: true,
		},
		{
			name: "err, dirty, pull request",
			state: &IssueMetadataState{
				dirty: true,
				Title: "a pull request",
				Type:  PRMetadata,
			},
			wantErrLine:      `X operation failed. To restore: pullpo pr create --recover .*testfile.*`,
			err:              true,
			wantPreservation: true,
		},
	}

	tempDir := t.TempDir()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.state == nil {
				tt.state = &IssueMetadataState{}
			}

			ios, _, _, errOut := iostreams.Test()

			tf, tferr := os.CreateTemp(tempDir, "testfile*")
			assert.NoError(t, tferr)
			defer tf.Close()

			ios.TempFileOverride = tf

			var err error
			if tt.err {
				err = errors.New("error during creation")
			}

			PreserveInput(ios, tt.state, &err)()

			_, err = tf.Seek(0, 0)
			assert.NoError(t, err)

			data, err := io.ReadAll(tf)
			assert.NoError(t, err)

			if tt.wantPreservation {
				//nolint:staticcheck // prefer exact matchers over ExpectLines
				test.ExpectLines(t, errOut.String(), tt.wantErrLine)
				preserved := &IssueMetadataState{}
				assert.NoError(t, json.Unmarshal(data, preserved))
				preserved.dirty = tt.state.dirty
				assert.Equal(t, preserved, tt.state)
			} else {
				assert.Equal(t, errOut.String(), "")
				assert.Equal(t, string(data), "")
			}
		})
	}
}
