package schema

import (
	"encoding/json"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// MonitorContract holds the schema definition for the MonitorContract entity.
type MonitorContract struct {
	ent.Schema
}

// Fields of the MonitorContract.
func (MonitorContract) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Comment("Primary key ID"),
		field.Int("chain_id").
			Comment("Chain ID (e.g., 1 for mainnet, 11155111 for Sepolia)"),
		field.String("address").
			MaxLen(42).
			Unique().
			Comment("Contract address"),
		field.String("name").
			MaxLen(64).
			Comment("Contract alias for easy identification"),
		field.JSON("abi", json.RawMessage{}).
			Comment("Contract ABI definition"),
		field.Int8("status").
			Default(1).
			Comment("Whether to enable monitoring (0: stop, 1: run)"),
	}
}

// Indexes of the MonitorContract.
func (MonitorContract) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("chain_id", "address").
			Unique(),
	}
}

// Edges of the MonitorContract.
func (MonitorContract) Edges() []ent.Edge {
	return []ent.Edge{
		// Define edges if needed, e.g., to events
	}
}
