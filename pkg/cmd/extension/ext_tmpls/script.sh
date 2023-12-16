#!/usr/bin/env bash
set -e

echo "Hello %[1]s!"

# Snippets to help get started:

# Determine if an executable is in the PATH
# if ! type -p ruby >/dev/null; then
#   echo "Ruby not found on the system" >&2
#   exit 1
# fi

# Pass arguments throupullpo to another command
# pullpo issue list "$@" -R cli/cli

# Using the pullpo api command to retrieve and format information
# QUERY='
#   query($endCursor: String) {
#     viewer {
#       repositories(first: 100, after: $endCursor) {
#         nodes {
#           nameWithOwner
#           stargazerCount
#         }
#       }
#     }
#   }
# '
# TEMPLATE='
#   {{- range $repo := .data.viewer.repositories.nodes -}}
#     {{- printf "name: %%s - stargazers: %%v\n" $repo.nameWithOwner $repo.stargazerCount -}}
#   {{- end -}}
# '
# exec pullpo api graphql -f query="${QUERY}" --paginate --template="${TEMPLATE}"
