package artifactory

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceArtifactoryLocalRpmRepository() *schema.Resource {
	var rpmLocalSchema = mergeSchema(baseLocalRepoSchema, map[string]*schema.Schema{
		"yum_root_depth": {
			Type:             schema.TypeInt,
			Optional:         true,
			Default:          0,
			ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
			Description: "The depth, relative to the repository's root folder, where RPM metadata is created. " +
				"This is useful when your repository contains multiple RPM repositories under parallel hierarchies. " +
				"For example, if your RPMs are stored under 'fedora/linux/$releasever/$basearch', specify a depth of 4.",
		},

		"calculate_yum_metadata": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},

		"enable_file_lists_indexing": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},

		"yum_group_file_names": {
			Type:             schema.TypeString,
			Optional:         true,
			Default:          "",
			ValidateDiagFunc: commaSeperatedList,
			Description: "A list of XML file names containing RPM group component definitions. Artifactory includes " +
				"the group definitions as part of the calculated RPM metadata, as well as automatically generating a " +
				"gzipped version of the group files, if required.",
		},
	})

	type RpmLocalRepositoryParams struct {
		LocalRepositoryBaseParams
		RootDepth               int    `hcl:"yum_root_depth" json:"yumRootDepth"`
		CalculateYumMetadata    bool   `hcl:"calculate_yum_metadata" json:"calculateYumMetadata"`
		EnableFileListsIndexing bool   `hcl:"enable_file_lists_indexing" json:"enableFileListsIndexing"`
		GroupFileNames          string `hcl:"yum_group_file_names" json:"yumGroupFileNames"`
	}

	unPackLocalRpmRepository := func(data *schema.ResourceData) (interface{}, string, error) {
		d := &ResourceData{ResourceData: data}
		repo := RpmLocalRepositoryParams{
			LocalRepositoryBaseParams: unpackBaseRepo("local", data, "rpm"),
			RootDepth:                 d.getInt("yum_root_depth", false),
			CalculateYumMetadata:      d.getBool("calculate_yum_metadata", false),
			EnableFileListsIndexing:   d.getBool("enable_file_lists_indexing", false),
			GroupFileNames:            d.getString("yum_group_file_names", false),
		}

		return repo, repo.Id(), nil
	}

	return mkResourceSchema(rpmLocalSchema, inSchema(rpmLocalSchema), unPackLocalRpmRepository, func() interface{} {
		return &RpmLocalRepositoryParams{
			LocalRepositoryBaseParams: LocalRepositoryBaseParams{
				PackageType: "rpm",
				Rclass:      "local",
			},
			RootDepth:               0,
			CalculateYumMetadata:    false,
			EnableFileListsIndexing: false,
			GroupFileNames:          "",
		}
	})
}
