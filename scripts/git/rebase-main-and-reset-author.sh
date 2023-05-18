#!/usr/bin/env bash

# doc: rebase origin/main and reset author of commits to current author

git fetch origin
git rebase origin/main --exec 'git commit --amend --reset-author --no-edit'
