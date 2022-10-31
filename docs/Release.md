# Release Process

This is the process to follow to make a new release. Similar to [Semantic](https://semver.org/) Version. The project refers to the respective components of this triple as <major>.<minor>.<patch>.
We automatically release a new release via `github action`, which includes the following sections:

## Pre-Release

- Check out to `main` branch, ensure it's up-to-date, and ensure you have a clean working directory.
- Use the following command to create a new tag and push:

```shell
git tag <new tag>
git push origin <new tag> -m "Release <new tag>"
```

> Note: The format of the release message must be in the formort: "Release <New tag>", For example: Release v0.2.0.

## On-Release

When you push your commit to origin main, Which triggers the `Auto Release` task of the github action. `Auto Release` includes this following steps:

- Checking that your commit message is in the required format("Release <New tag>").

If the above checks all passed, Now we start release a new release. 

- Create a branch with name release-<tag>
- Call build binary and upload.
- Call image build and push it to docker image registry.
- Call chart release and update it to remote helm repo.
- Create a new release.

OK, congratulations! We have successfully created a new release. This is the entire process of releasing a new release.