package pagerduty

import (
	"context"
	"fmt"
	"log"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type dataSourceIncidentWorkflowAction struct{ client *pagerduty.Client }

var _ datasource.DataSourceWithConfigure = (*dataSourceIncidentWorkflowAction)(nil)

func (*dataSourceIncidentWorkflowAction) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "pagerduty_incident_workflow_action"
}

func (*dataSourceIncidentWorkflowAction) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name":               schema.StringAttribute{Required: true},
			"version":            schema.Float64Attribute{Required: true},
			"id":                 schema.StringAttribute{Computed: true},
			"type":               schema.StringAttribute{Computed: true},
			"self":               schema.StringAttribute{Computed: true},
			"domain_name":        schema.StringAttribute{Computed: true},
			"package_name":       schema.StringAttribute{Computed: true},
			"function_name":      schema.StringAttribute{Computed: true},
			"description":        schema.StringAttribute{Computed: true},
			"action_type":        schema.StringAttribute{Computed: true},
			"tags":               schema.ListAttribute{ElementType: types.StringType, Computed: true},
			"search_keywords":    schema.ListAttribute{ElementType: types.StringType, Computed: true},
			"metadata":           schema.StringAttribute{Computed: true},
			"created_at":         schema.StringAttribute{Computed: true},
			"created_by_user_id": schema.StringAttribute{Computed: true},
		},
		Blocks: map[string]schema.Block{
			"inputs": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name":        schema.StringAttribute{Computed: true},
						"type":        schema.StringAttribute{Computed: true},
						"description": schema.StringAttribute{Computed: true},
						"required":    schema.BoolAttribute{Computed: true},
						"default":     schema.StringAttribute{Computed: true},
					},
				},
			},
			"outputs": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name":        schema.StringAttribute{Computed: true},
						"type":        schema.StringAttribute{Computed: true},
						"description": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *dataSourceIncidentWorkflowAction) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&d.client, req.ProviderData)...)
}

func (d *dataSourceIncidentWorkflowAction) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	log.Println("[INFO] Reading PagerDuty incident workflow action")

	var searchName types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("name"), &searchName)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var searchVersion types.Float64
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("version"), &searchVersion)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var found *pagerduty.IncidentWorkflowAction
	more := true
	cursor := ""

	log.Println("[CG] searching")
lookup:
	for more {
		response, err := d.client.ListIncidentWorkflowActions(ctx, pagerduty.ListIncidentWorkflowActionsOptions{
			Cursor: cursor,
			Limit:  100,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error reading PagerDuty incident workflow action %s", searchName),
				err.Error(),
			)
			return
		}

		log.Println("[CG] matching", len(response.IncidentWorkflowActions))
		for _, a := range response.IncidentWorkflowActions {
			log.Println("[CG]", a.Name, "| v", a.Version)
			if a.Name == searchName.ValueString() && a.Version == searchVersion.ValueFloat64() {
				found = &a
				break lookup
			}
		}

		cursor = response.NextCursor
		more = response.More
	}

	if found == nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to locate any incident workflow action with the name: %s", searchName),
			"",
		)
		return
	}

	log.Printf("[CG] %#v", found)

	tagsValues := make([]attr.Value, len(found.Tags))
	for i, tag := range found.Tags {
		tagsValues[i] = types.StringValue(tag)
	}

	searchKeywordsValues := make([]attr.Value, len(found.SearchKeywords))
	for i, keyword := range found.SearchKeywords {
		searchKeywordsValues[i] = types.StringValue(keyword)
	}

	inputValues := make([]attr.Value, len(found.Inputs))
	for i, input := range found.Inputs {
		inputValues[i] = types.ObjectValueMust(IncidentWorkflowActionInputObjectType.AttrTypes, map[string]attr.Value{
			"name":        types.StringValue(input.Name),
			"type":        types.StringValue(input.Type),
			"description": types.StringValue(input.Description),
			"required":    types.BoolValue(input.Required),
			"default":     types.StringValue(input.Default),
		})
	}

	outputValues := make([]attr.Value, len(found.Outputs))
	for i, output := range found.Outputs {
		outputValues[i] = types.ObjectValueMust(IncidentWorkflowActionOutputObjectType.AttrTypes, map[string]attr.Value{
			"name":        types.StringValue(output.Name),
			"type":        types.StringValue(output.Type),
			"description": types.StringValue(output.Description),
		})
	}

	model := dataSourceIncidentWorkflowActionModel{
		ID:              types.StringValue(found.ID),
		Name:            types.StringValue(found.Name),
		Type:            types.StringValue(found.Type),
		Self:            types.StringValue(found.Self),
		DomainName:      types.StringValue(found.DomainName),
		PackageName:     types.StringValue(found.PackageName),
		FunctionName:    types.StringValue(found.FunctionName),
		Version:         types.Float64Value(found.Version),
		Description:     types.StringValue(found.Description),
		ActionType:      types.StringValue(found.ActionType),
		Tags:            types.ListValueMust(types.StringType, tagsValues),
		SearchKeywords:  types.ListValueMust(types.StringType, searchKeywordsValues),
		Metadata:        types.StringValue(found.Metadata),
		CreatedAt:       types.StringValue(found.CreatedAt),
		CreatedByUserID: types.StringValue(found.CreatedByUserID),
		Inputs:          types.ListValueMust(IncidentWorkflowActionInputObjectType, inputValues),
		Outputs:         types.ListValueMust(IncidentWorkflowActionOutputObjectType, outputValues),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

type dataSourceIncidentWorkflowActionModel struct {
	ID              types.String  `tfsdk:"id"`
	Name            types.String  `tfsdk:"name"`
	Type            types.String  `tfsdk:"type"`
	Self            types.String  `tfsdk:"self"`
	DomainName      types.String  `tfsdk:"domain_name"`
	PackageName     types.String  `tfsdk:"package_name"`
	FunctionName    types.String  `tfsdk:"function_name"`
	Version         types.Float64 `tfsdk:"version"`
	Description     types.String  `tfsdk:"description"`
	ActionType      types.String  `tfsdk:"action_type"`
	Tags            types.List    `tfsdk:"tags"`
	SearchKeywords  types.List    `tfsdk:"search_keywords"`
	Metadata        types.String  `tfsdk:"metadata"`
	CreatedAt       types.String  `tfsdk:"created_at"`
	CreatedByUserID types.String  `tfsdk:"created_by_user_id"`
	Inputs          types.List    `tfsdk:"inputs"`
	Outputs         types.List    `tfsdk:"outputs"`
}

var IncidentWorkflowActionInputObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":        types.StringType,
		"type":        types.StringType,
		"description": types.StringType,
		"required":    types.BoolType,
		"default":     types.StringType,
	},
}

var IncidentWorkflowActionOutputObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":        types.StringType,
		"type":        types.StringType,
		"description": types.StringType,
	},
}
