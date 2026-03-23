#!/bin/bash
# caldav-cli skill installer
# Usage: bash skills/caldav-cli/install.sh /path/to/skills
# Or: curl -fsSL .../install.sh | bash -s -- /path/to/skills

set -e

# Configuration
SKILL_NAME="caldav-cli"
REPO_URL="https://raw.githubusercontent.com/ksinistr/caldav-cli/main"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Main installation
main() {
    # Get target directory
    SKILLS_DIR="${1:-}"

    if [ -z "$SKILLS_DIR" ]; then
        error "Usage: $0 /path/to/skills"
    fi

    # Create target directory
    if [ ! -d "$SKILLS_DIR" ]; then
        info "Creating skills directory: ${SKILLS_DIR}"
        mkdir -p "$SKILLS_DIR"
    fi

    # Target skill directory
    TARGET_DIR="${SKILLS_DIR}/${SKILL_NAME}"

    # Check if skill already exists
    if [ -d "$TARGET_DIR" ]; then
        warn "Skill directory already exists: ${TARGET_DIR}"
        read -p "Overwrite? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            info "Installation cancelled"
            exit 0
        fi
        rm -rf "$TARGET_DIR"
    fi

    info "Installing ${SKILL_NAME} skill to: ${TARGET_DIR}"

    # Create skill directory
    mkdir -p "$TARGET_DIR"

    # Download SKILL.md
    SKILL_URL="${REPO_URL}/skills/${SKILL_NAME}/SKILL.md"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$SKILL_URL" -o "${TARGET_DIR}/SKILL.md"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$SKILL_URL" -O "${TARGET_DIR}/SKILL.md"
    else
        error "Neither curl nor wget is available"
    fi

    # Verify installation
    if [ -f "${TARGET_DIR}/SKILL.md" ]; then
        info "Skill installed successfully!"
        info ""
        info "Skill location: ${TARGET_DIR}"
        info ""
        info "Ensure your agent system is configured to read skills from: ${SKILLS_DIR}"
    else
        error "Installation failed - SKILL.md not found"
    fi
}

main "$@"
