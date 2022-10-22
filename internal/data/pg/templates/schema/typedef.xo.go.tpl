{{- $t := .Data -}}
{{- if $t.Comment -}}
// {{ $t.Comment }}
{{- else -}}
// {{  $t.GoName  }}Q represents helper struct to access row of '{{ $t.SQLName }}'.
{{- end }}
type {{  $t.GoName  }}Q struct {
    db *pgdb.DB
}

// New{{  $t.GoName }}Q  - creates new instance
func New{{  $t.GoName }}Q(db *pgdb.DB) *{{  $t.GoName }}Q {
	return &{{  $t.GoName  }}Q{
		db,
	}
}

// {{  $t.GoName  }}Q  - creates new instance of {{  $t.GoName  }}Q
func (s Storage) {{  $t.GoName  }}Q() *{{  $t.GoName  }}Q {
    return New{{  $t.GoName }}Q(s.DB())
}

var cols{{  $t.GoName  }} = `{{ colnames $t.Fields }}`

// {{ func_name_context "Insert" }} inserts a {{ $t.GoName }} to the database.
{{ recv_context $t "Insert" }} {
{{ if $t.Manual }}
	// sql insert query, primary key must be provided
    {{ sqlstr "insert_manual" $t }}
    // run
    err := {{ db_prefix "ExecRaw" "" false $t }}
    return errors.Wrap(err, "failed to execute insert query")
{{- else -}}
    // insert (primary key generated and returned by database)
    {{ sqlstr "insert" $t }}
    // run
    {{ $pk := (index $t.PrimaryKeys 0)}}
    {{ $destName := (printf "&%s.%s" (short $t) $pk.GoName)}}
     err := {{ db_prefix "GetRaw" $destName true $t }}
     if err != nil {
        return errors.Wrap(err, "failed to execute insert")
     }

    return nil
{{end}}
}

{{ if context_both -}}
// Insert insert a {{ $t.GoName }} to the database.
{{ recv $t "Insert" }} {
	return q.InsertCtx(context.Background(), {{ short $t }})
}
{{- end }}

{{ if eq (len $t.Fields) (len $t.PrimaryKeys) -}}
// ------ NOTE: Update statements omitted due to lack of fields other than primary key ------
{{- else -}}
// {{ func_name_context "Update" }} updates a {{ $t.GoName }} in the database.
{{ recv_context $t "Update" }} {
    // update with {{ if driver "postgres" }}composite {{ end }}primary key
    {{ sqlstr "update" $t }}
    // run
    err := {{ db_update "ExecRaw" $t }}
    return errors.Wrap(err, "failed to execute update")
}

{{ if context_both -}}
// Update updates a {{ $t.GoName }} in the database.
{{ recv $t "Update" }} {
	return q.UpdateCtx(context.Background(), {{ short $t }})
}
{{- end }}


// {{ func_name_context "Upsert" }} performs an upsert for {{ $t.GoName }}.
{{ recv_context $t "Upsert" }} {
	// upsert
	{{ sqlstr "upsert" $t }}
	// run
	if err := {{ db_prefix "ExecRaw" "" false $t }}; err != nil {
		return errors.Wrap(err, "failed to execute upsert stmt")
	}
	return nil
}

{{ if context_both -}}
// Upsert performs an upsert for {{ $t.GoName }}.
{{ recv $t "Upsert" }} {
	return q.UpsertCtx(context.Background(), {{ short $t }})
}
{{- end -}}
{{- end }}

// {{ func_name_context "Delete" }} deletes the {{ $t.GoName }} from the database.
{{ recv_context $t "Delete" }} {
{{ if eq (len $t.PrimaryKeys) 1 -}}
	// delete with single primary key
	{{ sqlstr "delete" $t }}
	// run
	if err := {{ db "ExecRaw" "" (print (short $t) "." (index $t.PrimaryKeys 0).GoName) }}; err != nil {
		return errors.Wrap(err, "failed to exec delete stmt")
	}
{{- else -}}
	// delete with composite primary key
	{{ sqlstr "delete" $t }}
	// run
	if err := {{ db "ExecRaw" "" (names (print (short $t) ".") $t.PrimaryKeys) }}; err != nil {
		return errors.Wrap(err, "failed to exec delete stmt")
	}
{{- end }}
	return nil
}

{{ if context_both -}}
// Delete deletes the {{ $t.GoName }} from the database.
{{ recv $t "Delete" }} {
	return q.DeleteCtx(context.Background(), {{ short $t }})
}

{{- end -}}
