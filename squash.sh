#!/usr/bin/env bash

# Cross-platform compatibility checks
if [ -z "$BASH_VERSION" ]; then
    echo "❌ This script requires bash. Please run with: bash $0"
    exit 1
fi

# Use bash-specific features only if in bash
set -euo pipefail

# Usage: ./squash.sh [optional commit message] [--target-branch=branch] [--dry-run] [--force]

# Parse arguments
COMMIT_MSG=""
TARGET_BRANCH_OVERRIDE=""
DRY_RUN=false
FORCE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --target-branch=*)
            TARGET_BRANCH_OVERRIDE="${1#*=}"
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --force|-f)
            FORCE=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [commit-message] [--target-branch=branch] [--dry-run] [--force]"
            echo "  commit-message: Optional commit message for squashed commit"
            echo "  --target-branch: Specify target branch instead of auto-detection"
            echo "  --dry-run: Show what would be done without making changes"
            echo "  --force: Skip confirmation prompts"
            exit 0
            ;;
        *)
            if [[ -z "$COMMIT_MSG" ]]; then
                COMMIT_MSG="$1"
            else
                echo "❌ Unknown argument: $1"
                exit 1
            fi
            shift
            ;;
    esac
done

# Cross-platform date function
get_timestamp() {
    # Try GNU date first (Linux), then BSD date (macOS)
    if date --version >/dev/null 2>&1; then
        # GNU date (Linux)
        date +%Y%m%d-%H%M%S
    else
        # BSD date (macOS)
        date +%Y%m%d-%H%M%S
    fi
}

# Cross-platform read function with fallback
read_confirmation() {
    local prompt="$1"
    
    # Try different read options for compatibility
    if printf "%s" "$prompt"; then
        if read -n 1 -r REPLY 2>/dev/null; then
            echo  # Add newline after single character input
        elif read -r REPLY; then
            # Fallback for systems where -n doesn't work
            REPLY="${REPLY:0:1}"  # Take only first character
        else
            echo "Failed to read input"
            return 1
        fi
    fi
}

# --- 1. Validate environment ---
# Check if we're in a git repository
if ! git rev-parse --git-dir >/dev/null 2>&1; then
    echo "❌ Not in a git repository"
    exit 1
fi

# Check for required git commands
for cmd in "git rev-parse" "git status" "git for-each-ref" "git cherry" "git merge-base" "git log" "git reset" "git commit" "git push"; do
    if ! command -v ${cmd%% *} >/dev/null 2>&1; then
        echo "❌ Required command not found: ${cmd%% *}"
        exit 1
    fi
done

# --- 2. Validate working directory ---
if [[ -n "$(git status --porcelain)" ]]; then
    echo "❌ Working directory is not clean. Please commit or stash changes first."
    git status --short
    exit 1
fi

# --- 3. Get the current working branch ---
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

if [[ "$CURRENT_BRANCH" == "HEAD" ]]; then
    echo "❌ You are in a detached HEAD state. Please switch to a branch first."
    exit 1
fi

# --- 4. Determine the target branch ---
if [[ -n "$TARGET_BRANCH_OVERRIDE" ]]; then
    TARGET_BRANCH="$TARGET_BRANCH_OVERRIDE"
    # Validate the specified target branch exists
    if ! git show-ref --verify --quiet "refs/heads/$TARGET_BRANCH"; then
        echo "❌ Target branch '$TARGET_BRANCH' does not exist."
        exit 1
    fi
else
    # Auto-detect target branch
    POSSIBLE_TARGETS=$(git for-each-ref --format='%(refname:short)' refs/heads/ | grep -v "^${CURRENT_BRANCH}$")
    
    TARGET_BRANCH=""
    
    # Prefer common branch names first
    for preferred in "main" "master" "develop" "dev"; do
        if echo "$POSSIBLE_TARGETS" | grep -q "^${preferred}$"; then
            if git cherry "$preferred" "$CURRENT_BRANCH" | grep -q '^+'; then
                TARGET_BRANCH="$preferred"
                break
            fi
        fi
    done
    
    # If no preferred branch found, use the original logic
    if [[ -z "$TARGET_BRANCH" ]]; then
        for branch in $POSSIBLE_TARGETS; do
            if git cherry "$branch" "$CURRENT_BRANCH" | grep -q '^+'; then
                TARGET_BRANCH="$branch"
                break
            fi
        done
    fi
    
    if [[ -z "$TARGET_BRANCH" ]]; then
        echo "❌ Could not determine the target branch this branch was based on."
        echo "Available branches: $POSSIBLE_TARGETS"
        echo "Use --target-branch=<branch> to specify manually."
        exit 1
    fi
fi

# --- 5. Find the common ancestor commit ---
BASE=$(git merge-base "$TARGET_BRANCH" "$CURRENT_BRANCH")

# --- 6. Count commits to be squashed ---
COMMIT_COUNT=$(git rev-list --count "${BASE}..${CURRENT_BRANCH}")

if [[ "$COMMIT_COUNT" -eq 0 ]]; then
    echo "❌ No commits found to squash between '$TARGET_BRANCH' and '$CURRENT_BRANCH'."
    exit 1
fi

if [[ "$COMMIT_COUNT" -eq 1 ]]; then
    echo "ℹ️  Only one commit found. No squashing needed."
    exit 0
fi

# --- 7. Get the default commit message ---
if [[ -z "$COMMIT_MSG" ]]; then
    COMMIT_MSG=$(git log --reverse --format=%s "${BASE}..${CURRENT_BRANCH}" | head -n 1)
fi

# --- 8. Show what will be done ---
echo "🔍 Squash Summary:"
echo "   Current branch: $CURRENT_BRANCH"
echo "   Target branch:  $TARGET_BRANCH"
echo "   Commits to squash: $COMMIT_COUNT"
echo "   Base commit: ${BASE:0:7}"
echo "   New commit message: \"$COMMIT_MSG\""
echo

echo "📋 Commits to be squashed:"
git log --oneline "${BASE}..${CURRENT_BRANCH}"
echo

if [[ "$DRY_RUN" == true ]]; then
    echo "🔍 DRY RUN: Would squash $COMMIT_COUNT commits on '$CURRENT_BRANCH'."
    exit 0
fi

# --- 9. Confirmation prompt ---
if [[ "$FORCE" != true ]]; then
    read_confirmation "❓ Do you want to proceed with squashing? [y/N] "
    if [[ ! "$REPLY" =~ ^[Yy]$ ]]; then
        echo "❌ Aborted."
        exit 1
    fi
fi

# --- 10. Create backup branch ---
BACKUP_BRANCH="${CURRENT_BRANCH}-backup-$(get_timestamp)"
echo "💾 Creating backup branch: $BACKUP_BRANCH"
git branch "$BACKUP_BRANCH"

# --- 11. Perform the squash ---
echo "🔄 Squashing commits..."
git reset --soft "$BASE"
git commit -m "$COMMIT_MSG"

# --- 12. Push with safety ---
echo "📤 Pushing changes..."

# Check if remote exists and branch is tracked
if git ls-remote --exit-code origin >/dev/null 2>&1; then
    # Check if current branch exists on remote
    if git ls-remote --exit-code origin "$CURRENT_BRANCH" >/dev/null 2>&1; then
        # Use --force-with-lease for safety
        if git push --force-with-lease origin "$CURRENT_BRANCH"; then
            echo "✅ Successfully squashed '$CURRENT_BRANCH' into a single commit:"
            echo "   \"$COMMIT_MSG\""
            echo "💾 Backup branch created: $BACKUP_BRANCH"
            echo "   To restore: git reset --hard $BACKUP_BRANCH"
        else
            echo "❌ Push failed. Your local changes are preserved."
            echo "💾 Backup branch available: $BACKUP_BRANCH"
            echo "   This might happen if someone else pushed to the branch."
            echo "   You may need to force push: git push --force origin $CURRENT_BRANCH"
            exit 1
        fi
    else
        # Branch doesn't exist on remote, do a regular push
        if git push --set-upstream origin "$CURRENT_BRANCH"; then
            echo "✅ Successfully squashed '$CURRENT_BRANCH' into a single commit:"
            echo "   \"$COMMIT_MSG\""
            echo "💾 Backup branch created: $BACKUP_BRANCH"
        else
            echo "❌ Push failed. Your local changes are preserved."
            echo "💾 Backup branch available: $BACKUP_BRANCH"
            exit 1
        fi
    fi
else
    echo "⚠️  No remote 'origin' found. Local squash completed but not pushed."
    echo "✅ Successfully squashed '$CURRENT_BRANCH' into a single commit:"
    echo "   \"$COMMIT_MSG\""
    echo "💾 Backup branch created: $BACKUP_BRANCH"
    echo "   To push: git push --set-upstream origin $CURRENT_BRANCH"
fi
