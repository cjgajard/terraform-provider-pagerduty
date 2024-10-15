package optiontypes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type ObjectType struct {
	basetypes.ObjectType
}

var _ basetypes.ObjectTypable = ObjectType{}

func (o ObjectType) TerraformType(ctx context.Context) tftypes.Type {
	optional := map[string]struct{}{}
	attributeTypes := map[string]tftypes.Type{}
	for k, v := range o.AttrTypes {
		attributeTypes[k] = v.TerraformType(ctx)
		optional[k] = struct{}{}
	}
	return tftypes.Object{
		AttributeTypes:     attributeTypes,
		OptionalAttributes: optional,
	}
}

func (o ObjectType) Equal(t attr.Type) bool {
	// return o.ObjectType.Equal(t)
	return true
}

func (o ObjectType) String() string {
	return "optiontypes.ObjectType"
}
