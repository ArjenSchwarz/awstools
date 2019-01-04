workflow "Build and Deploy" {
    on = "push"
    resolves = ["Deploy"]
}

action "Build" {
    uses = "apex/actions/go@master"
    runs = "make"
    args = "build"
}

action "Deploy" {
    uses = "ArjenSchwarz/actions/github/release@master"
    needs = "Build"
    secrets = ["GITHUB_TOKEN"]
    args = "-delete"
    env = {
        ONLY_IN_BRANCH="master"
        SOURCE_PATH="dist"
        VERSION="latest"
    }
}