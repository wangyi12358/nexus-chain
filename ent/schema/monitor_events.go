package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// MonitorEvent holds the schema definition for the MonitorEvent entity.
type MonitorEvent struct {
	ent.Schema
}

// Fields of the MonitorEvent.
func (MonitorEvent) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Comment("Primary key ID"),
		field.UUID("contract_id", uuid.UUID{}).
			Comment("Foreign key to monitor_contracts.id"),
		field.String("event_name").
			MaxLen(64).
			Comment("Event name"),
		field.String("mq_routing_key").
			MaxLen(64).
			Comment("MQ routing key for this event"),
		field.Int8("status").
			Default(1).
			Comment("Whether to enable monitoring (0: stop, 1: run)"),
	}
}

// Indexes of the MonitorEvent.
func (MonitorEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("contract_id", "event_name").
			Unique(),
	}
}

// Edges of the MonitorEvent.
func (MonitorEvent) Edges() []ent.Edge {
	return []ent.Edge{
		// Define edge to MonitorContract
	}
}
