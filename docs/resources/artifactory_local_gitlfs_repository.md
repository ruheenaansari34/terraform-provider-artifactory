# Artifactory Local Gitlfs Repository Resource

Creates a local gitlfs repository.

## Example Usage

```hcl
resource "artifactory_local_gitlfs_repository" "terraform-local-test-gitlfs-repo" {
  key = "terraform-local-test-gitlfs-repo"
}
```

## Argument Reference

Arguments have a one to one mapping with the [JFrog API](https://www.jfrog.com/confluence/display/RTF/Repository+Configuration+JSON). The following arguments are supported:

* `key` - (Required) - the identity key of the repo
* `description` - (Optional)
* `notes` - (Optional)

Arguments for Gitlfs repository type closely match with arguments for Generic repository type.
