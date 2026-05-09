package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ChainNode holds the schema definition for the ChainNode entity.
type ChainNode struct {
	ent.Schema
}

// Fields of the ChainNode.
func (ChainNode) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Comment("Primary key ID"),
		field.Int("chain_id").
			Comment("Foreign key to chains.id"),
		field.String("node_type").
			MaxLen(16).
			Comment("Node type: rpc or ws"),
		field.String("name").
			MaxLen(64).
			Comment("Node alias"),
		field.String("url").
			MaxLen(255).
			Comment("Node endpoint URL"),
		field.Int("priority").
			Default(100).
			Comment("Lower value means higher priority"),
		field.Int8("status").
			Default(1).
			Comment("Whether to enable this node (0: stop, 1: run)"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Indexes of the ChainNode.
func (ChainNode) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("chain_id", "node_type", "status", "priority"),
		index.Fields("chain_id", "node_type", "url").
			Unique(),
	}
}
