package cloudiam

import (
	"context"
	"fmt"
	"log"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
)

func GetIAMSample() {
	ctx := context.Background()
	// Get credentials.
	client, err := google.DefaultClient(ctx, iam.CloudPlatformScope)
	if err != nil {
		log.Fatalf("google.DefaultClient: %v", err)
	}

	// Create the Cloud IAM service object.
	service, err := iam.New(client)
	if err != nil {
		log.Fatalf("iam.New: %v", err)
	}

	{
		role, err := service.Projects.Roles.Get("projects/soilworks-expt-01-266813/roles/200").Context(ctx).Do()
		if err != nil {
			log.Fatalf("iam.New: %v", err)
		}
		log.Println(role)
	}

	// Call the Cloud IAM Roles API.
	resp, err := service.Roles.List().Do()
	if err != nil {
		log.Fatalf("Roles.List: %v", err)
	}

	// Process the response.
	for _, role := range resp.Roles {
		log.Println("Tile: " + role.Title)
		log.Println("Name: " + role.Name)
		log.Println("Description: " + role.Description)
	}
}

func GetIAMPolicySample() {
	ctx := context.Background()
	// Get credentials.
	//client, err := google.DefaultClient(ctx, iam.CloudPlatformScope)
	//if err != nil {
	//	log.Fatalf("google.DefaultClient: %v", err)
	//}

	crm, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		panic(err)
	}
	p, err := crm.Projects.GetIamPolicy("soilworks-expt-01-266813", &cloudresourcemanager.GetIamPolicyRequest{}).Context(ctx).Do()
	if err != nil {
		panic(err)
	}
	for _, bind := range p.Bindings {
		fmt.Printf("%+v", bind)
	}
}
