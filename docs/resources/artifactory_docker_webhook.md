# Artifactory Docker Webhook Resource

Provides an Artifactory webhook resource. This can be used to register and manage Artifactory webhook subscription which enables you to be notified or notify other users when such events take place in Artifactory.

## Example Usage
.
```hcl
resource "artifactory_local_docker_v2_repository" "my-docker-local" {
  key = "my-docker-local"
}

resource "artifactory_docker_webhook" "docker-webhook" {
  key = "docker-webhook"
  event_types = ["pushed", "deleted", "promoted"]
  criteria {
    any_local = true
    any_remote = false
    repo_keys = [artifactory_local_docker_v2_repository.my-docker-local.key]
    include_patterns = ["foo/**"]
    exclude_patterns = ["bar/**"]
  }
  url = "http://tempurl.org/webhook"
  secret = "some-secret"
  proxy = "proxy-key"

  custom_http_headers = {
    header-1 = "value-1"
    header-2 = "value-2"
  }

  depends_on = [artifactory_local_docker_v2_repository.my-docker-local]
}
```

## Argument Reference

Arguments have a one to one mapping with the [JFrog Webhook API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API). The following arguments are supported:

The following arguments are supported:

* `key` - (Required) The identity key of the webhook. Must be between 2 and 200 characters. Cannot contain spaces.
* `description` - (Optional) Webhook description. Max length 1000 characters.
* `enabled` - (Optional) Status of webhook. Default to 'true'
* `event_types` - (Required) List of Events in Artifactory, Distribution, Release Bundle that function as the event trigger for the Webhook. Allow values: "pushed", "deleted", "promoted"
* `criteria` - (Required) Specifies where the webhook will be applied on which repositories.
  * `any_local` - (Required) Trigger on any local repo
  * `any_remote` - (Required) Trigger on any remote repo
  * `repo_keys` - (Required) Trigger on this list of repo keys
  * `include_patterns` - (Optional) Simple comma separated wildcard patterns for repository artifact paths (with no leading slash).\n Ant-style path expressions are supported (*, **, ?).\nFor example: "org/apache/**"
  * `exclude_patterns` - (Optional) Simple comma separated wildcard patterns for repository artifact paths (with no leading slash).\n Ant-style path expressions are supported (*, **, ?).\nFor example: "org/apache/**"
* `url` - (Required) Specifies the URL that the Webhook invokes. This will be the URL that Artifactory will send an HTTP POST request to.
* `secret` - (Optional) Secret authentication token that will be sent to the configured URL
* `proxy` - (Optional) Proxy key from Artifactory Proxies setting
* `custom_http_headers` - (Optional) Custom HTTP headers you wish to use to invoke the Webhook, comprise of key/value pair.
