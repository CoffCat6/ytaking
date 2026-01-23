Param(
  [string]$RepoPath = '.',
  [string]$PostsPath = 'data/posts.json',
  [string]$CommitMessage = "content: update posts.json",
  [switch]$Push
)

$ErrorActionPreference = 'Stop'

function Fail($msg) {
  Write-Error $msg
  exit 1
}

if (-not (Test-Path $RepoPath)) {
  Fail "RepoPath not found: $RepoPath"
}

Set-Location $RepoPath

if (-not (Test-Path $PostsPath)) {
  Fail "Posts file not found: $PostsPath"
}

# Check git
$gitVersion = git --version 2>$null
if (-not $gitVersion) {
  Fail "git not found in PATH"
}

# Ensure repo
$gitDir = git rev-parse --git-dir 2>$null
if (-not $gitDir) {
  Fail "Not a git repository: $RepoPath"
}

# Stage file
$add = git add $PostsPath 2>&1
if ($LASTEXITCODE -ne 0) {
  Fail "git add failed: $add"
}

# Commit if there is a change
$changes = git status --porcelain $PostsPath
if (-not $changes) {
  Write-Host "No changes in $PostsPath. Skip commit."
  exit 0
}

$commit = git commit -m $CommitMessage 2>&1
if ($LASTEXITCODE -ne 0) {
  Fail "git commit failed: $commit"
}

Write-Host $commit

if ($Push) {
  $pushOut = git push 2>&1
  if ($LASTEXITCODE -ne 0) {
    Fail "git push failed: $pushOut"
  }
  Write-Host $pushOut
}
