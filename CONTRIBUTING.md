# Contributing to Gödel Rescheduler

## Code of Conduct

Please do check our [Code of Conduct](CODE_OF_CONDUCT.md) before making contributions.

## Topics

* [Reporting security issues](#reporting-security-issues)
* [Reporting general issues](#reporting-general-issues)
* [Code and doc contribution](#code-and-doc-contribution)
* [Engage to help anything](#engage-to-help-anything)

## Reporting security issues

We take security issues seriously and discourage anyone to spread security issues.
If you find a security issue in Gödel Rescheduler, please do not discuss it in public and even do not open a public issue.
Instead, we encourage you to send us a private email to [Maintainers](MAINTAINERS.md) to report the security issue.

## Reporting general issues

Any user is welcome be a contributor. If you have any feedback for the project, feel free to open an issue.

Since Gödel Rescheduler development will be collaborated in a distributed manner, we appreciate **WELL-WRITTEN**, **DETAILED**, **EXPLICIT** issue reports.
To make communication more efficient, we suggest everyone to search if your issue is an existing one before filing a new issue. If you find it to be existing, please append your details in the issue comments.

There are a lot of cases for which you could open an issue:

* Bug report
* Feature request
* Performance issues
* Feature proposal
* Feature design
* Help wanted
* Doc incomplete
* Test improvement
* Any questions about the project, and so on

Please remind that when filing a new issue, do remove the sensitive data from your post.
Sensitive data could be password, secret key, network locations, private business data and so on.

## Code and doc contribution

Any action that may make Gödel Rescheduler better is encouraged. The action can be realized via a PR (short for pull request).

* If you find a typo, try to fix it!
* If you find a bug, try to fix it!
* If you find some redundant codes, try to remove them!
* If you find some test cases missing, try to add them!
* If you could enhance a feature, please **DO NOT** hesitate!
* If you find code implicit, try to add comments to make it clear!
* If you find tech debts, try to refactor them!
* If you find document incorrect, please fix that!

It is impossible to list them completely, we are looking forward to your pull requests.
Before submitting a PR, we suggest you could take a look at the PR rules here.

* [Workspace Preparation](#workspace-preparation)
* [Branch Definition](#branch-definition)
* [Commit Rules](#commit-rules)
* [PR Description](#pr-description)

### Workspace Preparation

We assume you have a GitHub ID already, then you could finish the preparation in the following steps:

1. **FORK** Gödel Rescheduler to your repository. To make this work, you just need to click the button `Fork` in top-right corner of [godel-rescheduler](https://github.com/kubewharf/godel-rescheduler) main page. Then you will end up with your repository in `https://github.com/<username>/godel-rescheduler`, in which `username` is your GitHub ID.
2. **CLONE** your own repository to develop locally. Use `git clone https://github.com/<username>/godel-rescheduler.git` to clone repository to your local machine. Then you can create new branches to finish the change you wish to make.
3. **Set Remote** upstream to be Gödel Rescheduler using the following two commands:

```
git remote add upstream https://github.com/kubewharf/godel-rescheduler.git
git remote set-url --push upstream no-pushing
```

With this remote setting, you can check your git remote configuration like this:

```
$ git remote -v
origin     https://github.com/<username>/godel-rescheduler.git (fetch)
origin     https://github.com/<username>/godel-rescheduler.git (push)
upstream   https://github.com/kubewharf/godel-rescheduler.git (fetch)
upstream   no-pushing (push)
```

With above, we can easily synchronize local branches with upstream branches.

### Branch Definition

Right now we assume every contribution via pull request is for the `main` branch in Gödel Rescheduler.
There are several other branches such as rc branches, release branches and backport branches.
Before officially releasing a version, we may check out a rc (release candidate) branch for more testings.
When officially releasing a version, there may be a release branch before tagging which will be deleted after tagging.
When backport some fixes to existing released version, we will check out backport branches.

### Commit Rules

In Gödel Rescheduler, we take two rules seriously for submitted PRs:

* [Commit Message](#commit-message)
* [Commit Content](#commit-content)

#### Commit Message

TODO

#### Commit Content

TODO

### PR Description

PR is the only way to make change to Gödel Rescheduler project. To help reviewers, we actually encourage contributors to make PR description as detailed as possible.

## Engage to help anything

GitHub is the primary place for Gödel Rescheduler contributors to collaborate. Although contributions via PR is an explicit way to help, we still call for any other types of helps.

* Reply to other's issues if you could;
* Help solve other user's problems;
* Help review other's PR design;
* Help review other's codes in PR;
* Discuss Gödel Rescheduler to make things clearer;
* Advocate Gödel Rescheduler technology beyond GitHub;
* Write blogs on Gödel Rescheduler, and so on.

In a word, **ANY HELP CAN BE A CONTRIBUTION.**
