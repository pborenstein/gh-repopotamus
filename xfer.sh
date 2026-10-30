#! /usr/bin/env bash
org="tepiton"
me="mimeo-sites"


# Transfer each repo; stop if any request fails.
while IFS= read -r repo; do
  echo "$org/$repo"
  gh api --method POST "repos/$org/$repo/transfer" \
    -f new_owner="$me" || break
done < sites 

