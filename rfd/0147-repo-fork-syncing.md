---
authors: Mike Jensen (mike.jensen@goteleport.com)
state: draft
---
 
# RFD 144 - Automatic Repo Syncing
 
## What
 
This RFD describes how we can sync a repo continuously and in an automatic way.  This can be used to provide private copies of a repo.
 
## Why
 
We currently face a few frictions which could be addressed through a more robust repo sync mechanism:
1. Our internal `teleport-private` fork used for security development must be manually updated
2. Some tooling (Dependabot) does not allow scanning for non-default branches, necessitating manual efforts to manage our release branches
3. Because much of the CI and scanning configuration is committed into the repo, we face noise from duplicate alerts when we use a repo for testing or as an internal fork
 
## Details
 
As such we need a solution that can provide the following:
* Automatic and regular updates to a repo to ensure it's in sync
* The ability to apply changes to the upstream repo (for example disable Actions that we don't want running on the repo copy, or enable additional scanning / actions)
 
### Branch Structure
 
* `main` - This will be the repos default branch.  This branch will contain the upstream history in addition to commits needed for the sync process or other custom changes on the repo copy
* `sync/upstream-[UPSTREAM_BRANCH_NAME]` - This branch will be an exact copy of the upstream branch.  If there is a need to create a PR in this repo the history should be based off this rather than `main`
* `sync/rebase` - This branch is where our custom changes will be committed.  It will be rebased on to the upstream changes before being committed into `main`
 
#### Branch Protections
 
The only branch that will be protected is the `sync/rebase` branch.  Other branches are expected to allow force pushes so they can be forcefully synced based off the upstream state.
 
### High Level Implementation
 
At a high level there are two primary components to this implementation:
1. A GitHub action which will be scheduled as well as triggered on changes to the `sync/rebase` branch.  This action will perform the actual rebase action and commit the results to the respective branches above.
2. A script that is invoked when the rebase can not be applied cleanly.  This can allow some custom rebase logic as a means to reduce how often the `sync/rebase` branch needs to be manually updated to avoid rebase conflicts.  A simple example is that workflow automations not desired in the repo copy may be removed in the `sync/rebase` branch.  That means that any changes to those committed workflows would result in a conflict.  However it can be safely resolved that any modified files removed in the `sync/rebase` branch can be resolved by simply removing.
 
At a high level the action will do the following steps:
1. Setup auth and any necessary repo specific configuration
2. Checkout the latest upstream contents
3. Fetch our custom changes
4. Push the exact copy branches `sync/upstream-[UPSTREAM_BRANCH_NAME]` to our fork
5. Rebase our `sync/rebase` branch on to the latest upstream changes, using the script for resolutions if necessary and possible
6. Force push the rebased changes to the `main` branch on our repo copy
 
### Teleport Security Scanning
 
This tooling will be leveraged to address the gaps in our release branch scanning by doing the following:
1. Create 3 repos (1 for each supported version): `teleport-sec_scan-1`, `teleport-sec_scan-2`, `teleport-sec_scan-3`.  `1` will reference our latest release branch, with each subsequent version inspecting a consecutively older version.
2. Setup the sync automation as documented above
3. Adjust Dependabot configuration to only notify for security updates on these branches
4. Adjust CodeQL Configuration to scan on these branches rather than trying to capture all branch coverage within just `teleport`
 
Step 4 is an optional addition, but since Dependabot results will be located here, it seems to co-locate other security reporting to these branches too.  This will also reduce the "Fixed" and "Reopened" flapping that can occur when changes are seen inconsistently between `master` and the release branches.
 
With the automation determining what branch it needs to sync to based off the upstream branches, there is the concern that as soon as the next version is created all repos will be updated.  This will leave a small gap of time where the oldest (soon to be deprecated) release is not being scanned.  This time window seems short enough to be tolerable, but an alternative would be to create a fourth repo to ensure consistent coverage.

