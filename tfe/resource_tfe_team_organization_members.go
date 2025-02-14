package tfe

import (
	"errors"
	"fmt"
	"log"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceTFETeamOrganizationMembers() *schema.Resource {
	return &schema.Resource{
		Create: resourceTFETeamOrganizationMembersCreate,
		Read:   resourceTFETeamOrganizationMembersRead,
		Update: resourceTFETeamOrganizationMembersUpdate,
		Delete: resourceTFETeamOrganizationMembersDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"team_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"organization_membership_ids": {
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceTFETeamOrganizationMembersCreate(d *schema.ResourceData, meta interface{}) error {
	tfeClient := meta.(*tfe.Client)

	// Get the team ID.
	teamID := d.Get("team_id").(string)

	var organizationMembershipIDs []string
	// Get all organization membership IDs
	for _, id := range d.Get("organization_membership_ids").(*schema.Set).List() {
		organizationMembershipIDs = append(organizationMembershipIDs, id.(string))
	}

	// Create a new options struct.
	options := tfe.TeamMemberAddOptions{
		OrganizationMembershipIDs: organizationMembershipIDs,
	}

	log.Printf("[DEBUG] Add organization memberships %v to team: %s", organizationMembershipIDs, teamID)
	err := tfeClient.TeamMembers.Add(ctx, teamID, options)
	if err != nil {
		return fmt.Errorf("Error adding organization memberships %v to team %s: %w", organizationMembershipIDs, teamID, err)
	}

	d.SetId(teamID)

	return nil
}

func resourceTFETeamOrganizationMembersRead(d *schema.ResourceData, meta interface{}) error {
	tfeClient := meta.(*tfe.Client)

	log.Printf("[DEBUG] Read organization memberships from team: %s", d.Id())
	organizationMemberships, err := tfeClient.TeamMembers.ListOrganizationMemberships(ctx, d.Id())
	if err != nil {
		if errors.Is(err, tfe.ErrResourceNotFound) {
			log.Printf("[DEBUG] Organization memberships for team %s do no longer exist", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading organization memberships from team %s: %w", d.Id(), err)
	}

	// Get all organization memberships and add them to object
	var organizationMembershipIDs []interface{}
	for _, membership := range organizationMemberships {
		organizationMembershipIDs = append(organizationMembershipIDs, membership.ID)
	}

	// Check if organization memberships were added at all
	if len(organizationMembershipIDs) > 0 {
		d.Set("team_id", d.Id())
		d.Set("organization_membership_ids", organizationMembershipIDs)
	} else {
		log.Printf("[DEBUG] Organization memberships for team %s do no longer exist", d.Id())
		d.SetId("")
	}

	return nil
}

func resourceTFETeamOrganizationMembersUpdate(d *schema.ResourceData, meta interface{}) error {
	tfeClient := meta.(*tfe.Client)

	var membershipIDsToDelete *schema.Set
	var membershipIDsToAdd *schema.Set

	if d.HasChange("organization_membership_ids") {
		oldMembershipIDs, newMembershipIDs := d.GetChange("organization_membership_ids")
		membershipIDsToDelete = oldMembershipIDs.(*schema.Set).Difference(newMembershipIDs.(*schema.Set))
		membershipIDsToAdd = newMembershipIDs.(*schema.Set).Difference(oldMembershipIDs.(*schema.Set))
	}

	// First add the new organization memberships.
	if membershipIDsToAdd.Len() > 0 {
		// Create a new options struct.
		options := tfe.TeamMemberAddOptions{}

		// Add all the organization memberships that need to be added.
		for _, id := range membershipIDsToAdd.List() {
			options.OrganizationMembershipIDs = append(options.OrganizationMembershipIDs, id.(string))
		}

		log.Printf("[DEBUG] Add organization memberships %v to team: %s", options.OrganizationMembershipIDs, d.Id())
		err := tfeClient.TeamMembers.Add(ctx, d.Id(), options)
		if err != nil {
			return fmt.Errorf("Error adding organization memberships to team %s: %w", d.Id(), err)
		}
	}

	// Then delete all the old users.
	if membershipIDsToDelete.Len() > 0 {
		// Create a new options struct.
		options := tfe.TeamMemberRemoveOptions{}

		// Add all the organization memberships that need to be removed.
		for _, id := range membershipIDsToDelete.List() {
			options.OrganizationMembershipIDs = append(options.OrganizationMembershipIDs, id.(string))
		}

		log.Printf("[DEBUG] Remove organization memberships %v from team: %s", options.OrganizationMembershipIDs, d.Id())
		err := tfeClient.TeamMembers.Remove(ctx, d.Id(), options)
		if err != nil {
			return fmt.Errorf("Error removing organization memberships from team %s: %w", d.Id(), err)
		}
	}

	return nil
}

func resourceTFETeamOrganizationMembersDelete(d *schema.ResourceData, meta interface{}) error {
	tfeClient := meta.(*tfe.Client)

	log.Printf("[DEBUG] Read organization memberships from team: %s", d.Id())
	organizationMemberships, err := tfeClient.TeamMembers.ListOrganizationMemberships(ctx, d.Id())
	if err != nil {
		if errors.Is(err, tfe.ErrResourceNotFound) {
			log.Printf("[DEBUG] Organization memberships for team %s do no longer exist", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading organization memberships from team %s: %w", d.Id(), err)
	}

	// Create a new options struct.
	options := tfe.TeamMemberRemoveOptions{}

	// Add all the users that need to be removed.
	for _, memberships := range organizationMemberships {
		options.OrganizationMembershipIDs = append(options.OrganizationMembershipIDs, memberships.ID)
	}

	log.Printf("[DEBUG] Remove organization memberships %v from team: %s", options.OrganizationMembershipIDs, d.Id())
	err = tfeClient.TeamMembers.Remove(ctx, d.Id(), options)
	if err != nil {
		return fmt.Errorf("Error removing organization membership %v to team %s: %w", options.OrganizationMembershipIDs, d.Id(), err)
	}

	return nil
}
