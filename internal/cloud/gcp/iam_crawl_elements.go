package gcp

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	cloudutils "github.com/coordimap/agent/internal/cloud/utils"
	"github.com/coordimap/agent/pkg/domain/agent"
	"github.com/coordimap/agent/pkg/domain/gcp"
	gcpModel "github.com/coordimap/agent/pkg/domain/gcp"
	"github.com/coordimap/agent/pkg/utils"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
	gcpiam "google.golang.org/api/iam/v1"
	run "google.golang.org/api/run/v1"
	"google.golang.org/api/storage/v1"
)

func (gcpCrawler *gcpCrawler) getIAMElements(crawlTime time.Time) ([]*agent.Element, error) {
	allIAMElements := []*agent.Element{}

	client, errClient := cloudresourcemanager.NewService(context.Background(), gcpCrawler.clientOpts...)
	if errClient != nil {
		return nil, fmt.Errorf("could not create cloud resource manager client because %v", errClient)
	}

	project, errProject := client.Projects.Get(gcpCrawler.ConfiguredProjectID).Do()
	if errProject != nil {
		return nil, fmt.Errorf("could not retrieve project because %v", errProject)
	}

	projectInternalID := cloudutils.CreateGCPInternalName(gcpCrawler.scopeID, "", gcpModel.TypeProject, gcpCrawler.ConfiguredProjectID)
	projectName := project.Name
	if projectName == "" {
		projectName = gcpCrawler.ConfiguredProjectID
	}

	projectElem, errProjectElem := utils.CreateElement(project, projectName, projectInternalID, gcpModel.TypeProject, agent.StatusNoStatus, "", crawlTime)
	if errProjectElem == nil {
		allIAMElements = append(allIAMElements, projectElem)
	}

	iamAdminClient, errIAMAdminClient := gcpiam.NewService(context.Background(), gcpCrawler.clientOpts...)
	if errIAMAdminClient != nil {
		return allIAMElements, fmt.Errorf("could not create IAM admin client because %v", errIAMAdminClient)
	}

	serviceAccountElems, errServiceAccounts := gcpCrawler.getServiceAccounts(projectInternalID, iamAdminClient, crawlTime)
	if errServiceAccounts == nil {
		allIAMElements = append(allIAMElements, serviceAccountElems...)
	}

	customRoleElems, errCustomRoles := gcpCrawler.getCustomRoles(projectInternalID, iamAdminClient, crawlTime)
	if errCustomRoles == nil {
		allIAMElements = append(allIAMElements, customRoleElems...)
	}

	policy, errPolicy := client.Projects.GetIamPolicy(gcpCrawler.ConfiguredProjectID, &cloudresourcemanager.GetIamPolicyRequest{}).Do()
	if errPolicy != nil {
		return allIAMElements, fmt.Errorf("could not retrieve project IAM policy because %v", errPolicy)
	}

	predefinedRoleCache := map[string]bool{}
	allIAMElements = append(allIAMElements, gcpCrawler.buildIAMPolicyElements(projectInternalID, gcpModel.TypeProject, policy.Bindings, iamAdminClient, predefinedRoleCache, crawlTime)...)

	bucketIAMElems, errBucketIAMElems := gcpCrawler.getBucketIAMElements(iamAdminClient, predefinedRoleCache, crawlTime)
	if errBucketIAMElems == nil {
		allIAMElements = append(allIAMElements, bucketIAMElems...)
	}

	cloudRunIAMElems, errCloudRunIAMElems := gcpCrawler.getCloudRunIAMElements(iamAdminClient, predefinedRoleCache, crawlTime)
	if errCloudRunIAMElems == nil {
		allIAMElements = append(allIAMElements, cloudRunIAMElems...)
	}

	return allIAMElements, nil
}

func (gcpCrawler *gcpCrawler) getServiceAccounts(projectInternalID string, iamClient *gcpiam.Service, crawlTime time.Time) ([]*agent.Element, error) {
	allElems := []*agent.Element{}
	parent := fmt.Sprintf("projects/%s", gcpCrawler.ConfiguredProjectID)
	serviceAccounts, errServiceAccounts := iamClient.Projects.ServiceAccounts.List(parent).Do()
	if errServiceAccounts != nil {
		return nil, errServiceAccounts
	}

	for _, serviceAccount := range serviceAccounts.Accounts {
		serviceAccountInternalID := cloudutils.CreateGCPInternalName(gcpCrawler.scopeID, "", gcpModel.TypeServiceAccount, sanitizeIAMName(serviceAccount.Email))
		serviceAccountElem, errServiceAccountElem := utils.CreateElement(serviceAccount, serviceAccount.Email, serviceAccountInternalID, gcpModel.TypeServiceAccount, agent.StatusNoStatus, "", crawlTime)
		if errServiceAccountElem == nil {
			allElems = append(allElems, serviceAccountElem)
		}

		utils.AddRelationship(&allElems, projectInternalID, serviceAccountInternalID, agent.ParentChildTypeRelation, crawlTime)
	}

	return allElems, nil
}

func (gcpCrawler *gcpCrawler) getCustomRoles(projectInternalID string, iamClient *gcpiam.Service, crawlTime time.Time) ([]*agent.Element, error) {
	allElems := []*agent.Element{}
	parent := fmt.Sprintf("projects/%s", gcpCrawler.ConfiguredProjectID)
	rolesResp, errRoles := iamClient.Projects.Roles.List(parent).Do()
	if errRoles != nil {
		return nil, errRoles
	}

	for _, role := range rolesResp.Roles {
		roleName := role.Name
		if roleName == "" {
			roleName = role.Title
		}

		roleInternalID := cloudutils.CreateGCPInternalName(gcpCrawler.scopeID, "", gcpModel.TypeIAMRole, sanitizeIAMName(roleName))
		roleElem, errRoleElem := utils.CreateElement(role, roleName, roleInternalID, gcpModel.TypeIAMRole, agent.StatusNoStatus, role.Stage, crawlTime)
		if errRoleElem == nil {
			allElems = append(allElems, roleElem)
		}

		utils.AddRelationship(&allElems, projectInternalID, roleInternalID, agent.ParentChildTypeRelation, crawlTime)
	}

	return allElems, nil
}

func (gcpCrawler *gcpCrawler) getBucketIAMElements(iamClient *gcpiam.Service, predefinedRoleCache map[string]bool, crawlTime time.Time) ([]*agent.Element, error) {
	allElems := []*agent.Element{}
	client, errClient := storage.NewService(context.Background(), gcpCrawler.clientOpts...)
	if errClient != nil {
		return nil, errClient
	}

	buckets, errBuckets := client.Buckets.List(gcpCrawler.ConfiguredProjectID).Do()
	if errBuckets != nil {
		return nil, errBuckets
	}

	for _, bucket := range buckets.Items {
		bucketInternalID := cloudutils.CreateGCPInternalName(gcpCrawler.scopeID, strings.ToLower(bucket.Location), gcpModel.TypeBucket, bucket.Name)
		policy, errPolicy := client.Buckets.GetIamPolicy(bucket.Name).OptionsRequestedPolicyVersion(3).Do()
		if errPolicy != nil {
			continue
		}

		allElems = append(allElems, gcpCrawler.buildStorageIAMPolicyElements(bucketInternalID, gcpModel.TypeBucket, policy.Bindings, iamClient, predefinedRoleCache, crawlTime)...)
	}

	return allElems, nil
}

func (gcpCrawler *gcpCrawler) getCloudRunIAMElements(iamClient *gcpiam.Service, predefinedRoleCache map[string]bool, crawlTime time.Time) ([]*agent.Element, error) {
	allElems := []*agent.Element{}
	client, errClient := run.NewService(context.Background(), gcpCrawler.clientOpts...)
	if errClient != nil {
		return nil, errClient
	}

	parent := fmt.Sprintf("projects/%s/locations/-", gcpCrawler.ConfiguredProjectID)
	services, errServices := client.Projects.Locations.Services.List(parent).Do()
	if errServices != nil {
		return nil, errServices
	}

	for _, service := range services.Items {
		resourceName := service.Metadata.Name
		serviceInternalID := cloudutils.CreateGCPInternalName(gcpCrawler.scopeID, "", gcp.TypeCloudRun, service.Metadata.Name)
		policy, errPolicy := client.Projects.Locations.Services.GetIamPolicy(resourceName).OptionsRequestedPolicyVersion(3).Do()
		if errPolicy != nil {
			continue
		}

		allElems = append(allElems, gcpCrawler.buildRunIAMPolicyElements(serviceInternalID, gcpModel.TypeCloudRun, policy.Bindings, iamClient, predefinedRoleCache, crawlTime)...)
	}

	return allElems, nil
}

func (gcpCrawler *gcpCrawler) buildIAMPolicyElements(resourceInternalID, resourceType string, bindings []*cloudresourcemanager.Binding, iamClient *gcpiam.Service, predefinedRoleCache map[string]bool, crawlTime time.Time) []*agent.Element {
	allElems := []*agent.Element{}
	for _, binding := range bindings {
		bindingMembers := append([]string(nil), binding.Members...)
		sort.Strings(bindingMembers)

		bindingID := buildIAMBindingInternalID(gcpCrawler.scopeID, resourceInternalID, binding.Role, bindingMembers, binding.Condition)
		bindingElem, errBindingElem := utils.CreateElement(binding, binding.Role, bindingID, gcpModel.TypeIAMBinding, agent.StatusNoStatus, "", crawlTime)
		if errBindingElem == nil {
			allElems = append(allElems, bindingElem)
		}

		utils.AddRelationship(&allElems, resourceInternalID, bindingID, agent.ParentChildTypeRelation, crawlTime)

		roleInternalID := cloudutils.CreateGCPInternalName(gcpCrawler.scopeID, "", gcpModel.TypeIAMRole, sanitizeIAMName(binding.Role))
		if isPredefinedIAMRole(binding.Role) && !predefinedRoleCache[binding.Role] {
			role, errRole := iamClient.Roles.Get(binding.Role).Do()
			if errRole == nil {
				roleElem, errRoleElem := utils.CreateElement(role, role.Name, roleInternalID, gcpModel.TypeIAMRole, agent.StatusNoStatus, role.Stage, crawlTime)
				if errRoleElem == nil {
					allElems = append(allElems, roleElem)
					predefinedRoleCache[binding.Role] = true
				}
			}
		}

		utils.AddRelationship(&allElems, bindingID, roleInternalID, agent.ParentChildTypeRelation, crawlTime)

		for _, member := range bindingMembers {
			principalType, principalValue, ok := parseIAMPrincipal(member)
			if !ok {
				continue
			}

			principalInternalID := cloudutils.CreateGCPInternalName(gcpCrawler.scopeID, "", principalType, sanitizeIAMName(principalValue))
			principalElem, errPrincipalElem := utils.CreateElement(map[string]string{
				"member": member,
				"kind":   principalType,
				"value":  principalValue,
			}, principalValue, principalInternalID, principalType, agent.StatusNoStatus, "", crawlTime)
			if errPrincipalElem == nil {
				allElems = append(allElems, principalElem)
			}

			utils.AddRelationship(&allElems, bindingID, principalInternalID, agent.ParentChildTypeRelation, crawlTime)
		}
	}

	return allElems
}

func (gcpCrawler *gcpCrawler) buildStorageIAMPolicyElements(resourceInternalID, resourceType string, bindings []*storage.PolicyBindings, iamClient *gcpiam.Service, predefinedRoleCache map[string]bool, crawlTime time.Time) []*agent.Element {
	crmBindings := make([]*cloudresourcemanager.Binding, 0, len(bindings))
	for _, binding := range bindings {
		crmBinding := &cloudresourcemanager.Binding{
			Role:    binding.Role,
			Members: binding.Members,
		}
		if binding.Condition != nil {
			crmBinding.Condition = &cloudresourcemanager.Expr{
				Title:       binding.Condition.Title,
				Description: binding.Condition.Description,
				Expression:  binding.Condition.Expression,
				Location:    binding.Condition.Location,
			}
		}
		crmBindings = append(crmBindings, crmBinding)
	}

	return gcpCrawler.buildIAMPolicyElements(resourceInternalID, resourceType, crmBindings, iamClient, predefinedRoleCache, crawlTime)
}

func (gcpCrawler *gcpCrawler) buildRunIAMPolicyElements(resourceInternalID, resourceType string, bindings []*run.Binding, iamClient *gcpiam.Service, predefinedRoleCache map[string]bool, crawlTime time.Time) []*agent.Element {
	crmBindings := make([]*cloudresourcemanager.Binding, 0, len(bindings))
	for _, binding := range bindings {
		crmBinding := &cloudresourcemanager.Binding{
			Role:    binding.Role,
			Members: binding.Members,
		}
		if binding.Condition != nil {
			crmBinding.Condition = &cloudresourcemanager.Expr{
				Title:       binding.Condition.Title,
				Description: binding.Condition.Description,
				Expression:  binding.Condition.Expression,
				Location:    binding.Condition.Location,
			}
		}
		crmBindings = append(crmBindings, crmBinding)
	}

	return gcpCrawler.buildIAMPolicyElements(resourceInternalID, resourceType, crmBindings, iamClient, predefinedRoleCache, crawlTime)
}
