package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Chain holds the schema definition for the Chain entity.
type Chain struct {
	ent.Schema
}

// Fields of the Chain.
func (Chain) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Comment("Primary key ID"),
		field.Int("chain_id").
			Positive().
			Comment("EVM chain ID"),
		field.String("name").
			MaxLen(64).
			Comment("Chain name"),
		field.String("native_symbol").
			MaxLen(16).
			Comment("Native token symbol"),
		field.Int("confirmations").
			Default(6).
			Comment("Safe confirmation blocks"),
		field.Int("scan_batch_size").
			Default(1000).
			Comment("Historical scan batch size"),
		field.Int8("status").
			Default(1).
			Comment("Whether to enable this chain (0: stop, 1: run)"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Indexes of the Chain.
func (Chain) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("chain_id").
			Unique(),
	}
}
