package artifactory

import (
	"context"
	"gopkg.in/yaml.v2"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type Backup struct {
	Key                    string   `xml:"key" yaml:"key"`
	CronExp                string   `xml:"cronExp" yaml:"cronExp"`
	Enabled                bool     `xml:"enabled" yaml:"enabled"`
	RetentionPeriodHours   int      `xml:"retentionPeriodHours" yaml:"retentionPeriodHours"`
	ExcludedRepositories   []string `xml:"excludedRepositories>repositoryRef" yaml:"excludedRepositories"`
	CreateArchive          bool     `xml:"createArchive" yaml:"createArchive"`
	ExcludeNewRepositories bool     `xml:"excludeNewRepositories" yaml:"excludeNewRepositories"`
	SendMailOnError        bool     `xml:"sendMailOnError" yaml:"sendMailOnError"`
}

type Backups struct {
	BackupArr []Backup `xml:"backups>backup" yaml:"backup"`
}

func resourceArtifactoryBackup() *schema.Resource {
	var backupSchema = map[string]*schema.Schema{
		"key": {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
			Description:      `(Required) Backup config name.`,
		},
		"enabled": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: `(Optional) Flag to enable or disable the backup config. Default value is "true".`,
		},
		"cron_exp": {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validateCron),
			Description:      `(Required) Cron expression to control the backup frequency.`,
		},
		"retention_period_hours": {
			Type:             schema.TypeInt,
			Optional:         true,
			Default:          168,
			ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
			Description:      `(Optional) The number of hours to keep a backup before Artifactory will clean it up to free up disk space. Applicable only to non-incremental backups. Default value is 168 hours ie: 7 days.`,
		},
		"excluded_repositories": {
			Type:        schema.TypeList,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: `(Optional) list of excluded repositories from the backup. Default is empty list.`,
		},
		"create_archive": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: `(Optional) If set to true, backups will be created within a Zip archive (Slow and CPU intensive). Default value is 'false'`,
		},
		"exclude_new_repositories": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: `(Optional) When set to true, new repositories will not be automatically added to the backup. Default value is 'false'.`,
		},
		"send_mail_on_error": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: `(Optional) If set to true, all Artifactory administrators will be notified by email if any problem is encountered during backup. Default value is 'true'.`,
		},
	}
	var findBackup = func(backups *Backups, key string) Backup {
		for _, iterBackup := range backups.BackupArr {
			if iterBackup.Key == key {
				return iterBackup
			}
		}
		return Backup{}
	}
	var filterBackups = func(backups *Backups, key string) map[string]Backup {
		var filteredMap = map[string]Backup{}
		for _, iterBackup := range backups.BackupArr {
			if iterBackup.Key != key {
				filteredMap[iterBackup.Key] = iterBackup
			}
		}
		return filteredMap
	}
	var resourceBackupRead = func(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		backups := &Backups{}
		backup := unpackBackup(d)

		_, err := m.(*resty.Client).R().SetResult(&backups).Get("artifactory/api/system/configuration")
		if err != nil {
			return diag.Errorf("failed to retrieve data from API: /artifactory/api/system/configuration during Read")
		}

		matchedBackup := findBackup(backups, backup.Key)
		packer := universalPack(
			allHclPredicate(
				noClass, schemaHasKey(backupSchema),
			),
		)
		return diag.FromErr(packer(&matchedBackup, d))
	}

	var resourceBackupUpdate = func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		unpackedBackup := unpackBackup(d)

		/* EXPLANATION FOR BELOW CONSTRUCTION USAGE.
		There is a difference in xml structure usage between GET and PATCH calls of API: /artifactory/api/system/configuration.
		GET call structure has "backups -> backup -> Array of backup config blocks".
		PATCH call structure has "backups -> Name/Key of backup that is being patched -> config block of the backup being patched".
		Since the Name/Key is dynamic string, following nested map of string structs are constructed to match the usage of PATCH call.
		*/
		var constructBody = map[string]map[string]Backup{}
		constructBody["backups"] = map[string]Backup{}
		constructBody["backups"][unpackedBackup.Key] = unpackedBackup
		content, err := yaml.Marshal(&constructBody)

		if err != nil {
			return diag.FromErr(err)
		}

		err = sendConfigurationPatch(content, m)
		if err != nil {
			return diag.FromErr(err)
		}

		// we should only have one backup config resource, using same id
		d.SetId(unpackedBackup.Key)
		return resourceBackupRead(ctx, d, m)
	}

	var resourceBackupDelete = func(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		backups := &Backups{}
		rsrcBackup := unpackBackup(d)

		response, err := m.(*resty.Client).R().SetResult(&backups).Get("artifactory/api/system/configuration")
		if err != nil {
			return diag.FromErr(err)
		}
		if response.IsError() {
			return diag.Errorf("got error response for API: /artifactory/api/system/configuration request during Read. Response:%#v", response)
		}

		/* EXPLANATION FOR BELOW CONSTRUCTION USAGE.
		There is a difference in xml structure usage between GET and PATCH calls of API: /artifactory/api/system/configuration.
		GET call structure has "backups -> backup -> Array of backup config blocks".
		PATCH call structure has "backups -> Name/Key of backup that is being patched -> config block of the backup being patched".
		Since the Name/Key is dynamic string, following nested map of string structs are constructed to match the usage of PATCH call.
		*/
		var restoreBackups = map[string]map[string]Backup{}
		restoreBackups["backups"] = filterBackups(backups, rsrcBackup.Key)

		var clearAllBackupConfigs = `
backups: ~
`
		err = sendConfigurationPatch([]byte(clearAllBackupConfigs), m)
		if err != nil {
			return diag.FromErr(err)
		}

		restoreRestOfBackups, err := yaml.Marshal(&restoreBackups)
		if err != nil {
			return diag.FromErr(err)
		}

		err = sendConfigurationPatch([]byte(restoreRestOfBackups), m)
		if err != nil {
			return diag.FromErr(err)
		}
		return nil
	}

	return &schema.Resource{
		UpdateContext: resourceBackupUpdate,
		CreateContext: resourceBackupUpdate,
		DeleteContext: resourceBackupDelete,
		ReadContext:   resourceBackupRead,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema:      backupSchema,
		Description: "Provides an Artifactory backup config resource. This resource configuration corresponds to backup config block in system configuration XML (REST endpoint: artifactory/api/system/configuration). Manages the automatic and periodic backups of the entire Artifactory instance",
	}
}

func unpackBackup(s *schema.ResourceData) Backup {
	d := &ResourceData{s}
	backup := Backup{
		Key:                    d.getString("key", false),
		Enabled:                d.getBool("enabled", false),
		CronExp:                d.getString("cron_exp", false),
		RetentionPeriodHours:   d.getInt("retention_period_hours", false),
		CreateArchive:          d.getBool("create_archive", false),
		ExcludeNewRepositories: d.getBool("exclude_new_repositories", false),
		SendMailOnError:        d.getBool("send_mail_on_error", false),
		ExcludedRepositories:   d.getList("excluded_repositories"),
	}
	return backup
}
