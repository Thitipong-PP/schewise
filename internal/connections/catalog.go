// SPDX-License-Identifier: Apache-2.0
package connections

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
)

// Every relation read here belongs to pg_catalog; no user relation is queried,
// sampled, analyzed, or counted. Definitions and comments are untrusted facts.
const catalogSQL = `SELECT n.nspname,c.relname,
 CASE c.relkind WHEN 'r' THEN 'table' WHEN 'v' THEN 'view' WHEN 'm' THEN 'materialized_view' WHEN 'p' THEN 'partitioned_table' END,
 pg_catalog.obj_description(c.oid,'pg_class'),c.reltuples::float8,
 COALESCE((SELECT jsonb_agg(jsonb_build_object(
  'name',a.attname,'ordinal',a.attnum,'data_type',pg_catalog.format_type(a.atttypid,a.atttypmod),
  'nullable',NOT(a.attnotnull OR t.typnotnull),
  'default_expression',CASE WHEN a.attgenerated='' THEN pg_catalog.pg_get_expr(d.adbin,d.adrelid) END,
  'identity',CASE a.attidentity WHEN 'a' THEN 'always' WHEN 'd' THEN 'by_default' ELSE '' END,
  'generated',CASE a.attgenerated WHEN 's' THEN 'stored' WHEN 'v' THEN 'virtual' ELSE '' END,
  'generated_expression',CASE WHEN a.attgenerated<>'' THEN pg_catalog.pg_get_expr(d.adbin,d.adrelid) END,
  'comment',pg_catalog.col_description(c.oid,a.attnum)) ORDER BY a.attnum)
  FROM pg_catalog.pg_attribute a JOIN pg_catalog.pg_type t ON t.oid=a.atttypid
  LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum
  WHERE a.attrelid=c.oid AND a.attnum>0 AND NOT a.attisdropped),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(jsonb_build_object(
  'name',k.conname,'kind',CASE k.contype WHEN 'p' THEN 'primary_key' WHEN 'u' THEN 'unique' WHEN 'f' THEN 'foreign_key' WHEN 'c' THEN 'check' END,
  'columns',ARRAY(SELECT a.attname FROM unnest(k.conkey) WITH ORDINALITY x(num,ord) JOIN pg_catalog.pg_attribute a ON a.attrelid=k.conrelid AND a.attnum=x.num ORDER BY x.ord),
  'definition',pg_catalog.pg_get_constraintdef(k.oid,false),
  'reference',CASE WHEN k.contype='f' THEN jsonb_build_object('schema',rn.nspname,'table',rc.relname,
    'columns',ARRAY(SELECT a.attname FROM unnest(k.confkey) WITH ORDINALITY x(num,ord) JOIN pg_catalog.pg_attribute a ON a.attrelid=k.confrelid AND a.attnum=x.num ORDER BY x.ord)) END,
  'on_update',CASE WHEN k.contype='f' THEN CASE k.confupdtype WHEN 'a' THEN 'no_action' WHEN 'r' THEN 'restrict' WHEN 'c' THEN 'cascade' WHEN 'n' THEN 'set_null' WHEN 'd' THEN 'set_default' END ELSE '' END,
  'on_delete',CASE WHEN k.contype='f' THEN CASE k.confdeltype WHEN 'a' THEN 'no_action' WHEN 'r' THEN 'restrict' WHEN 'c' THEN 'cascade' WHEN 'n' THEN 'set_null' WHEN 'd' THEN 'set_default' END ELSE '' END,
  'deferrable',k.condeferrable,'initially_deferred',k.condeferred,'validated',k.convalidated,
  'comment',pg_catalog.obj_description(k.oid,'pg_constraint')) ORDER BY k.conname COLLATE "C")
  FROM pg_catalog.pg_constraint k LEFT JOIN pg_catalog.pg_class rc ON rc.oid=k.confrelid
  LEFT JOIN pg_catalog.pg_namespace rn ON rn.oid=rc.relnamespace
  WHERE k.conrelid=c.oid AND k.contype IN ('p','u','f','c')),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(jsonb_build_object(
  'name',ic.relname,
  'columns',ARRAY(SELECT pg_catalog.pg_get_indexdef(i.indexrelid,x,false) FROM generate_series(1,i.indnkeyatts) x ORDER BY x),
  'included_columns',ARRAY(SELECT pg_catalog.pg_get_indexdef(i.indexrelid,x,false) FROM generate_series(i.indnkeyatts+1,i.indnatts) x ORDER BY x),
  'unique',i.indisunique,'primary',i.indisprimary,'valid',i.indisvalid,'method',am.amname,
  'definition',pg_catalog.pg_get_indexdef(i.indexrelid,0,false),'predicate',pg_catalog.pg_get_expr(i.indpred,i.indrelid),
  'comment',pg_catalog.obj_description(i.indexrelid,'pg_class')) ORDER BY ic.relname COLLATE "C")
  FROM pg_catalog.pg_index i JOIN pg_catalog.pg_class ic ON ic.oid=i.indexrelid
  JOIN pg_catalog.pg_am am ON am.oid=ic.relam WHERE i.indrelid=c.oid),'[]'::jsonb)
 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
 WHERE c.relkind IN ('r','v','m','p') AND n.nspname<>'information_schema' AND n.nspname!~'^pg_'`

func normalize(content *Content) {
	if content.Schemas == nil {
		content.Schemas = []Schema{}
	}
	sort.Slice(content.Schemas, func(i, j int) bool { return content.Schemas[i].Name < content.Schemas[j].Name })
	for si := range content.Schemas {
		schema := &content.Schemas[si]
		if schema.Tables == nil {
			schema.Tables = []Table{}
		}
		sort.Slice(schema.Tables, func(i, j int) bool { return schema.Tables[i].Name < schema.Tables[j].Name })
		for ti := range schema.Tables {
			table := &schema.Tables[ti]
			if table.Columns == nil {
				table.Columns = []Column{}
			}
			if table.Constraints == nil {
				table.Constraints = []Constraint{}
			}
			if table.Indexes == nil {
				table.Indexes = []Index{}
			}
			sort.Slice(table.Columns, func(i, j int) bool { return table.Columns[i].Ordinal < table.Columns[j].Ordinal })
			sort.Slice(table.Constraints, func(i, j int) bool { return table.Constraints[i].Name < table.Constraints[j].Name })
			sort.Slice(table.Indexes, func(i, j int) bool { return table.Indexes[i].Name < table.Indexes[j].Name })
		}
	}
}
func encodeContent(content *Content) ([]byte, string, error) {
	normalize(content)
	data, err := json.Marshal(content)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(data)
	return data, hex.EncodeToString(sum[:]), nil
}

func introspect(ctx context.Context, conn *pgx.Conn) (Content, Stats, error) {
	content := Content{Schemas: []Schema{}}
	stats := Stats{Estimates: []RowEstimate{}, ObservedAt: time.Now().UTC()}
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return content, stats, targetError(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	rows, err := tx.Query(ctx, `SELECT nspname,pg_catalog.obj_description(oid,'pg_namespace') FROM pg_catalog.pg_namespace WHERE nspname<>'information_schema' AND nspname!~'^pg_' ORDER BY nspname COLLATE "C"`)
	if err != nil {
		return content, stats, targetError(err)
	}
	indices := map[string]int{}
	for rows.Next() {
		var schema Schema
		schema.Tables = []Table{}
		if err := rows.Scan(&schema.Name, &schema.Comment); err != nil {
			rows.Close()
			return content, stats, targetError(err)
		}
		indices[schema.Name] = len(content.Schemas)
		content.Schemas = append(content.Schemas, schema)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return content, stats, targetError(err)
	}
	rows, err = tx.Query(ctx, catalogSQL)
	if err != nil {
		return content, stats, targetError(err)
	}
	for rows.Next() {
		var schema string
		var table Table
		var estimate float64
		var columns, constraints, indexes []byte
		if err := rows.Scan(&schema, &table.Name, &table.Kind, &table.Comment, &estimate, &columns, &constraints, &indexes); err != nil {
			rows.Close()
			return content, stats, targetError(err)
		}
		if err := json.Unmarshal(columns, &table.Columns); err != nil {
			rows.Close()
			return content, stats, err
		}
		if err := json.Unmarshal(constraints, &table.Constraints); err != nil {
			rows.Close()
			return content, stats, err
		}
		if err := json.Unmarshal(indexes, &table.Indexes); err != nil {
			rows.Close()
			return content, stats, err
		}
		if stats.Tables >= 10000 || stats.Columns+len(table.Columns) > 100000 {
			rows.Close()
			return content, stats, &TargetError{"schema_too_large", "The schema exceeds the current introspection size limit."}
		}
		index := indices[schema]
		content.Schemas[index].Tables = append(content.Schemas[index].Tables, table)
		stats.Tables++
		stats.Columns += len(table.Columns)
		stats.Constraints += len(table.Constraints)
		stats.Indexes += len(table.Indexes)
		stats.Estimates = append(stats.Estimates, RowEstimate{Schema: schema, Table: table.Name, Rows: estimate})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return content, stats, targetError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return content, stats, targetError(err)
	}
	stats.Schemas = len(content.Schemas)
	normalize(&content)
	sort.Slice(stats.Estimates, func(i, j int) bool {
		a, b := stats.Estimates[i], stats.Estimates[j]
		if a.Schema != b.Schema {
			return a.Schema < b.Schema
		}
		return a.Table < b.Table
	})
	return content, stats, nil
}
