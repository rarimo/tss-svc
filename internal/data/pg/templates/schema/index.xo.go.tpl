{{ $i := .Data }}
// {{ func_name_context $i }} retrieves a row from '{{ schema $i.Table.SQLName }}' as a {{ $i.Table.GoName }}.
//
// Generated from index '{{ $i.SQLName }}'.
{{ func_context $i }} {
	// query
	{{ sqlstr "index" $i }}
	// run
	if isForUpdate {
        sqlstr += " for update"
	}
{{- if $i.IsUnique }}
	var res data.{{ $i.Table.GoName }}
	err := {{ db "GetRaw" "&res"  $i }}
	if err != nil {
	    if errors.Cause(err)  == sql.ErrNoRows {
	        return nil, nil
	    }

	    return nil, errors.Wrap(err, "failed to exec select")
	}

	return &res, nil
{{- else }}
	var res []data.{{ $i.Table.GoName }}
    err := {{ db "SelectRaw" "&res"  $i }}
    if err != nil {
        return nil, errors.Wrap(err, "failed to exec select")
    }

    return res, nil
{{- end }}
}

{{ if context_both -}}
// {{ func_name $i }} retrieves a row from '{{ schema $i.Table.SQLName }}' as a {{ $i.Table.GoName }}.
//
// Generated from index '{{ $i.SQLName }}'.
{{ func $i }} {
	return q.{{ func_name_context $i }}({{ names "" "context.Background()" $i "isForUpdate" }})
}
{{- end }}

