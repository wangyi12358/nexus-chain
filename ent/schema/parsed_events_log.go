package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ParsedEventsLog holds the schema definition for the ParsedEventsLog entity.
type ParsedEventsLog struct {
	ent.Schema
}

// Annotations of the ParsedEventsLog.
func (ParsedEventsLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "parsed_events_log"},
	}
}

// Fields of the ParsedEventsLog.
func (ParsedEventsLog) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Comment("Primary key ID"),
		field.String("uid").
			MaxLen(128).
			Comment("Business UID"),
		field.Int("chain_id").
			Comment("Chain ID"),
		field.UUID("event_id", uuid.UUID{}).
			Comment("Foreign key to monitor_events.id"),
		field.Int64("block_number").
			Comment("Block number"),
		field.String("tx_hash").
			MinLen(66).
			MaxLen(66).
			Comment("Transaction hash"),
		field.Int64("log_index").
			NonNegative().
			Comment("Log index inside the transaction"),
		field.JSON("parsed_data", map[string]interface{}{}).
			Comment("Parsed business data"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("Insert time"),
	}
}

// Indexes of the ParsedEventsLog.
func (ParsedEventsLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("chain_id", "block_number"),
		index.Fields("event_id", "block_number"),
		index.Fields("chain_id", "tx_hash", "log_index").
			Unique(),
	}
}
