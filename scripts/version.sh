#!/bin/bash

VERSION_PATH=version

parse_version() {
	local version=$(cat $VERSION_PATH)
	IFS='.' read -ra PARTS <<< "$version"

	MAJOR=${PARTS[0]}
    MINOR=${PARTS[1]}
    PATCH=${PARTS[2]}
}

write_version() {
    echo "$1" > "$VERSION_PATH"
    echo "Version: $1"
}

create_tag() {
    git tag "$(cat $VERSION_PATH)"
}

bump_major() {
    parse_version
    MAJOR=$((MAJOR + 1))
    MINOR=0
    PATCH=0
}

bump_minor() {
    parse_version
    MINOR=$((MINOR + 1))
    PATCH=0
}

bump_patch() {
    parse_version
    PATCH=$((PATCH + 1))
}

parse_version
if [ $# -eq 0 ]; then
    bump_patch
elif [ $# -eq 1 ]; then
    case "$1" in
        "major")
            bump_major
            ;;
        "minor")
            bump_minor
            ;;
        "patch")
            bump_patch
            ;;
        *)
            echo "Unknown args"
            exit 1
    esac
else
	echo "Invalid args length"
	exit 1
fi

write_version "$MAJOR.$MINOR.$PATCH"
create_tag
