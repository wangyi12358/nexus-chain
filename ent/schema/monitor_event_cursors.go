package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// MonitorEventCursor holds the schema definition for the MonitorEventCursor entity.
type MonitorEventCursor struct {
	ent.Schema
}

// Fields of the MonitorEventCursor.
func (MonitorEventCursor) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Comment("Primary key ID"),
		field.UUID("event_id", uuid.UUID{}).
			Comment("Foreign key to monitor_events.id"),
		field.Int64("scan_last_block").
			Default(0).
			Comment("Last successfully scanned block"),
		field.Time("last_scanned_at").
			Nillable().
			Optional().
			Comment("Last successful scan time"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Indexes of the MonitorEventCursor.
func (MonitorEventCursor) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("event_id").
			Unique(),
	}
}
