// Package schema 定义 Ent ORM 的数据库 schema。
package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Upstream 定义上游实例实体的 schema。
//
// 上游是一个独立的平台连接实体，维护与 sub2api/newapi 实例的
// 登录会话（JWT token / session cookie），并定期同步分组和余额元数据。
// 上游与 Account 完全独立——账号可以引用上游获取 API key，但上游本身不是账号。
type Upstream struct {
	ent.Schema
}

func (Upstream) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "upstreams"},
	}
}

func (Upstream) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (Upstream) Fields() []ent.Field {
	return []ent.Field{
		// 基本信息
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.String("platform").
			MaxLen(20).
			NotEmpty().
			Comment("sub2api | newapi"),
		field.String("base_url").
			MaxLen(500).
			NotEmpty(),
		field.String("email").
			MaxLen(200).
			NotEmpty(),
		field.String("password").
			NotEmpty().
			Sensitive(). // 不在日志/序列化中输出
			SchemaType(map[string]string{dialect.Postgres: "text"}),

		// Session 状态（由 UpstreamSyncWorker 每分钟维护）
		field.String("access_token").
			Optional().Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.String("refresh_token").
			Optional().Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Time("expires_at").
			Optional().Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("session_cookie").
			Optional().Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Int64("upstream_user_id").
			Optional().Nillable(),

		// 同步元数据（由 SyncUpstream 更新）
		field.JSON("groups", []map[string]any{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Default([]map[string]any{}),
		field.Float("balance").
			Default(0).
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,6)"}),
		field.String("health").
			MaxLen(20).
			Default("pending").
			Comment("pending | ok | error | syncing"),
		field.String("health_msg").
			Optional().Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Time("last_synced_at").
			Optional().Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (Upstream) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("platform"),
		index.Fields("health"),
		index.Fields("deleted_at"),
	}
}
