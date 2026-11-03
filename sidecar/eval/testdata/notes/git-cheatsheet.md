# Git cheat sheet

Undo the last commit but keep the changes: `git reset --soft HEAD~1`. Stash work in progress: `git stash push -m wip`.
Rewrite local commits before a pull request: `git rebase -i origin/main`. Find the commit that introduced a bug:
`git bisect start`, then mark commits good or bad. Delete merged branches: `git branch --merged | xargs git branch -d`.
