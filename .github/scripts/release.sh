#!/usr/bin/env bash
set -euo pipefail

release_main() {
  : "${RELEASE_TAG:?}" "${GITHUB_REPOSITORY:?}"
  [[ "$RELEASE_TAG" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$ ]] || {
    echo 'Invalid version tag' >&2; return 1;
  }
  case "${1:-}" in
    prepare)
      test "$(git rev-parse "refs/tags/$RELEASE_TAG^{commit}")" = "$(git rev-parse HEAD)"
      git merge-base --is-ancestor HEAD refs/remotes/origin/main
      local notes="$RUNNER_TEMP/vrpc-release-notes.md"
      awk -v prefix="## [${RELEASE_TAG#v}] - " '
        index($0, prefix) == 1 {
          date = substr($0, length(prefix) + 1)
          if (date !~ /^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]$/) exit 1
          found = 1; next
        }
        found && /^## / { exit }
        found { print; if ($0 ~ /[^[:space:]]/) body = 1 }
        END { if (!found || !body) exit 1 }
      ' CHANGELOG.md > "$notes"
      ;;
    publish)
      test -s "$RUNNER_TEMP/vrpc-release-notes.md"
      local release prerelease=false
      release=$(gh api --paginate --slurp "repos/$GITHUB_REPOSITORY/releases?per_page=100" |
        jq -ce --arg tag "$RELEASE_TAG" '[.[][] | select(.tag_name == $tag)] |
          if length == 0 then {} elif length == 1 then .[0] else error("Duplicate release") end')
      if [[ "$(jq -r .draft <<< "$release")" == false ]]; then
        echo 'Release already published; leaving it unchanged'
        return
      fi
      [[ "$RELEASE_TAG" != *-* ]] || prerelease=true
      if [[ "$(jq -r '.tag_name // empty' <<< "$release")" == "" ]]; then
        gh release create "$RELEASE_TAG" --repo "$GITHUB_REPOSITORY" --verify-tag \
          --draft --prerelease="$prerelease" --title "$RELEASE_TAG" --notes-file "$RUNNER_TEMP/vrpc-release-notes.md"
      fi
      gh release edit "$RELEASE_TAG" --repo "$GITHUB_REPOSITORY" --verify-tag --draft=false
      ;;
    *) echo 'Usage: release.sh prepare|publish' >&2; return 1 ;;
  esac
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then release_main "$@"; fi
