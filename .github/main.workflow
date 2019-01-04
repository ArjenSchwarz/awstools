workflow "Build and Deploy" {
    on = "push"
    resolves = ["Deploy"]
}

action "Build" {
    uses = "apex/actions/go@master"
    runs = "make"
    args = "build"
}

action "Package" {
    needs = "Build"
    uses = "ArjenSchwarz/actions/utils/zip@master"
    runs = "make"
    args = "package"
}

action "Deploy" {
    uses = "ArjenSchwarz/actions/github/release@master"
    needs = "Package"
    secrets = ["GITHUB_TOKEN"]
    args = "-delete"
    env = {
        ONLY_IN_BRANCH="master"
        SOURCE_PATH="dist"
        VERSION="latest"
    }
}