package artifactory

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/go-resty/resty/v2"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type GetReplicationConfig struct {
	RepoKey                string               `json:"-"`
	CronExp                string               `json:"cronExp,omitempty"`
	EnableEventReplication bool                 `json:"enableEventReplication,omitempty"`
	Replications           []getReplicationBody `json:"replications,omitempty"`
}

type UpdateReplicationConfig struct {
	RepoKey                string                  `json:"-"`
	CronExp                string                  `json:"cronExp,omitempty"`
	EnableEventReplication bool                    `json:"enableEventReplication,omitempty"`
	Replications           []updateReplicationBody `json:"replications,omitempty"`
}

var replicationSchemaCommon = map[string]*schema.Schema{
	"repo_key": {
		Type:     schema.TypeString,
		Required: true,
	},
	"cron_exp": {
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validateCron,
	},
	"enable_event_replication": {
		Type:     schema.TypeBool,
		Optional: true,
		Computed: true,
	},
}

var repMultipleSchema = map[string]*schema.Schema{
	"replications": {
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: replicationSchema,
		},
	},
}
var replicationSchema = map[string]*schema.Schema{
	"url": {
		Type:         schema.TypeString,
		Optional:     true,
		ForceNew:     true,
		ValidateFunc: validation.IsURLWithHTTPorHTTPS,
	},
	"socket_timeout_millis": {
		Type:         schema.TypeInt,
		Optional:     true,
		Computed:     true,
		ValidateFunc: validation.IntAtLeast(0),
	},
	"username": {
		Type:     schema.TypeString,
		Optional: true,
	},
	"password": {
		Type:      schema.TypeString,
		Optional:  true,
		Sensitive: true,
	},
	"enabled": {
		Type:     schema.TypeBool,
		Optional: true,
		Computed: true,
	},
	"sync_deletes": {
		Type:     schema.TypeBool,
		Optional: true,
		Computed: true,
	},
	"sync_properties": {
		Type:     schema.TypeBool,
		Optional: true,
		Computed: true,
	},
	"sync_statistics": {
		Type:     schema.TypeBool,
		Optional: true,
		Computed: true,
	},
	"path_prefix": {
		Type:     schema.TypeString,
		Optional: true,
	},
	"proxy": {
		Type:     schema.TypeString,
		Optional: true,
		Description: "Proxy key from Artifactory Proxies setting",
	},
}

func resourceArtifactoryReplicationConfig() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceReplicationConfigCreate,
		ReadContext:   resourceReplicationConfigRead,
		UpdateContext: resourceReplicationConfigUpdate,
		DeleteContext: resourceReplicationDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchema(replicationSchemaCommon, repMultipleSchema),
		DeprecationMessage: "This resource has been deprecated in favour of the more explicitly name" +
			"artifactory_push_replication resource.",
	}
}

func unpackReplicationConfig(s *schema.ResourceData) UpdateReplicationConfig {
	d := &ResourceData{s}
	replicationConfig := new(UpdateReplicationConfig)

	repo := d.getString("repo_key", false)

	if v, ok := d.GetOkExists("replications"); ok {
		arr := v.([]interface{})

		tmp := make([]updateReplicationBody, 0, len(arr))
		replicationConfig.Replications = tmp

		for i, o := range arr {
			if i == 0 {
				replicationConfig.RepoKey = repo
				replicationConfig.CronExp = d.getString("cron_exp", false)
				replicationConfig.EnableEventReplication = d.getBool("enable_event_replication", false)
			}

			m := o.(map[string]interface{})

			var replication updateReplicationBody

			replication.RepoKey = repo

			if v, ok = m["url"]; ok {
				replication.URL = v.(string)
			}

			if v, ok = m["socket_timeout_millis"]; ok {
				replication.SocketTimeoutMillis = v.(int)
			}

			if v, ok = m["username"]; ok {
				replication.Username = v.(string)
			}

			if v, ok = m["enabled"]; ok {
				replication.Enabled = v.(bool)
			}

			if v, ok = m["sync_deletes"]; ok {
				replication.SyncDeletes = v.(bool)
			}

			if v, ok = m["sync_properties"]; ok {
				replication.SyncProperties = v.(bool)
			}

			if v, ok = m["sync_statistics"]; ok {
				replication.SyncStatistics = v.(bool)
			}

			if prefix, ok := m["path_prefix"]; ok {
				replication.PathPrefix = prefix.(string)
			}

			if _, ok := m["proxy"]; ok {
				replication.Proxy = handleResetWithNonExistantValue(d, fmt.Sprintf("replications.%d.proxy", i))
			}

			if pass, ok := m["password"]; ok {
				replication.Password = pass.(string)
			}

			replicationConfig.Replications = append(replicationConfig.Replications, replication)
		}
	}

	return *replicationConfig
}

func packReplicationConfig(replicationConfig *GetReplicationConfig, d *schema.ResourceData) diag.Diagnostics {
	var errors []error
	setValue := mkLens(d)

	setValue("repo_key", replicationConfig.RepoKey)
	setValue("cron_exp", replicationConfig.CronExp)
	errors = setValue("enable_event_replication", replicationConfig.EnableEventReplication)

	if replicationConfig.Replications != nil {
		var replications []map[string]interface{}
		for _, repo := range replicationConfig.Replications {
			replication := make(map[string]interface{})

			replication["url"] = repo.URL
			replication["socket_timeout_millis"] = repo.SocketTimeoutMillis
			replication["username"] = repo.Username
			replication["password"] = getMD5Hash(repo.Password)
			replication["enabled"] = repo.Enabled
			replication["sync_deletes"] = repo.SyncDeletes
			replication["sync_properties"] = repo.SyncProperties
			replication["sync_statistics"] = repo.SyncStatistics
			replication["path_prefix"] = repo.PathPrefix
			replication["proxy"] = repo.ProxyRef
			replications = append(replications, replication)
		}

		errors = setValue("replications", replications)
	}
	if errors != nil && len(errors) > 0 {
		return diag.Errorf("failed to pack replication config %q", errors)
	}

	return nil
}

func resourceReplicationConfigCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	replicationConfig := unpackReplicationConfig(d)

	_, err := m.(*resty.Client).R().SetBody(replicationConfig).Put("artifactory/api/replications/multiple/" + replicationConfig.RepoKey)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(replicationConfig.RepoKey)
	return resourceReplicationConfigRead(ctx, d, m)
}

func resourceReplicationConfigRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*resty.Client)
	var replications []getReplicationBody
	_, err := c.R().SetResult(&replications).Get("artifactory/api/replications/" + d.Id())

	if err != nil {
		return diag.FromErr(err)
	}

	repConfig := GetReplicationConfig{
		RepoKey:      d.Id(),
		Replications: replications,
	}
	if len(replications) > 0 {
		repConfig.EnableEventReplication = replications[0].EnableEventReplication
		repConfig.CronExp = replications[0].CronExp
	}
	return packReplicationConfig(&repConfig, d)
}

func resourceReplicationConfigUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	replicationConfig := unpackReplicationConfig(d)

	_, err := m.(*resty.Client).R().SetBody(replicationConfig).Post("/api/replications/" + d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(replicationConfig.RepoKey)

	return resourceReplicationConfigRead(ctx, d, m)
}
