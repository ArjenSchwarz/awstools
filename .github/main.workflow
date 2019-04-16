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

action "Production Filter" {
  uses = "actions/bin/filter@707718ee26483624de00bd146e073d915139a3d8"
  needs = ["Package"]
  args = "branch master"
}

action "Deploy" {
    uses = "ArjenSchwarz/actions/github/release@master"
    needs = "Production Filter"
    secrets = ["GITHUB_TOKEN"]
    args = "-delete"
    env = {
        SOURCE_PATH="dist"
        VERSION="latest"
    }
}