package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Autumn-27/artex/researchkit"
)

var ErrResearchConflict = errors.New("research record changed or already exists; reload before saving")

type ResearchRecord struct {
	Kind      string                       `json:"kind"`
	TaskID    int64                        `json:"task_id,string"`
	Key       string                       `json:"key"`
	Title     string                       `json:"title"`
	Version   int64                        `json:"version"`
	Content   researchkit.WorkspaceContent `json:"content"`
	UpdatedAt time.Time                    `json:"updated_at"`
}

func scanResearch(row interface{ Scan(...any) error }) (*ResearchRecord, error) {
	var r ResearchRecord
	var raw []byte
	err := row.Scan(&r.Kind, &r.TaskID, &r.Key, &r.Title, &r.Version, &raw, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(raw, &r.Content); err != nil {
		return nil, err
	}
	return &r, nil
}

const researchColumns = `kind,scope_id,key,title,version,content,updated_at`

func (d *DB) ResearchGet(kind string, taskID int64, key string) (*ResearchRecord, error) {
	return scanResearch(d.QueryRow(`SELECT `+researchColumns+` FROM research_records WHERE kind=$1 AND scope_id=$2 AND key=$3`, kind, taskID, key))
}
func (d *DB) ResearchList(kind string, taskID int64, q string) ([]*ResearchRecord, error) {
	rows, err := d.Query(`SELECT `+researchColumns+` FROM research_records WHERE kind=$1 AND scope_id=$2 AND ($3='' OR strpos(lower(title||' '||key||' '||content::text),lower($3))>0) ORDER BY updated_at DESC,key LIMIT 200`, kind, taskID, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*ResearchRecord{}
	for rows.Next() {
		r, err := scanResearch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func (d *DB) ResearchSave(r ResearchRecord) (*ResearchRecord, error) {
	if err := researchkit.ValidateWorkspace(r.Kind, r.Key, r.Title, r.TaskID, r.Content); err != nil {
		return nil, err
	}
	if r.Version < 0 {
		return nil, errors.New("version must be zero for create or the current positive revision")
	}
	raw, err := json.Marshal(r.Content)
	if err != nil {
		return nil, err
	}
	tx, err := d.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var row *ResearchRecord
	if r.Version == 0 {
		row, err = scanResearch(tx.QueryRow(`INSERT INTO research_records(kind,task_id,key,title,content) VALUES($1,NULLIF($2,0),$3,$4,$5) ON CONFLICT(kind,scope_id,key) DO NOTHING RETURNING `+researchColumns, r.Kind, r.TaskID, r.Key, r.Title, raw))
	} else {
		row, err = scanResearch(tx.QueryRow(`UPDATE research_records SET title=$4,content=$5,version=version+1,updated_at=now() WHERE kind=$1 AND scope_id=$2 AND key=$3 AND version=$6 RETURNING `+researchColumns, r.Kind, r.TaskID, r.Key, r.Title, raw, r.Version))
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrResearchConflict
	}
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(`INSERT INTO research_revisions(record_id,version,title,content) SELECT id,version,title,content FROM research_records WHERE kind=$1 AND scope_id=$2 AND key=$3`, r.Kind, r.TaskID, r.Key)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return row, nil
}
func (d *DB) ResearchDelete(kind string, taskID int64, key string, version int64) error {
	if version < 1 {
		return ErrResearchConflict
	}
	result, err := d.Exec(`DELETE FROM research_records WHERE kind=$1 AND scope_id=$2 AND key=$3 AND version=$4`, kind, taskID, key, version)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrResearchConflict
	}
	return nil
}
func (d *DB) ResearchHistory(kind string, taskID int64, key string) ([]*ResearchRecord, error) {
	rows, err := d.Query(`SELECT r.kind,r.scope_id,r.key,h.title,h.version,h.content,h.created_at FROM research_revisions h JOIN research_records r ON r.id=h.record_id WHERE r.kind=$1 AND r.scope_id=$2 AND r.key=$3 ORDER BY h.version DESC LIMIT 100`, kind, taskID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*ResearchRecord{}
	for rows.Next() {
		r, err := scanResearch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SeedResearchRole inserts a native conversational role atomically, preserving edits.
func (d *DB) SeedResearchRole(role researchkit.ReviewRole, tools []string, skills []string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRow(`INSERT INTO agents(key,name,description,role,builtin,enabled,max_turns,web_search,interactive_shell) VALUES($1,$2,$3,'assistant',false,true,24,false,false) ON CONFLICT(key) DO NOTHING RETURNING id`, role.Key, role.Name, role.Description).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var promptID int64
	err = tx.QueryRow(`INSERT INTO agent_prompts(agent_id,version,template_text,note,updated_by) VALUES($1,1,$2,'TYAN review adaptation','seed') RETURNING id`, id, role.Prompt).Scan(&promptID)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE agents SET current_prompt_id=$2 WHERE id=$1`, id, promptID); err != nil {
		return err
	}
	for _, name := range skills {
		if _, err = tx.Exec(`INSERT INTO agent_skill_visibility(agent_id,skill_name,enabled) VALUES($1,$2,true) ON CONFLICT DO NOTHING`, id, name); err != nil {
			return err
		}
	}
	for _, name := range tools {
		if _, err = tx.Exec(`UPDATE tools SET agents=agents || to_jsonb($1::text) WHERE key=$2 AND NOT agents @> jsonb_build_array($1::text)`, role.Key, name); err != nil {
			return err
		}
	}
	return tx.Commit()
}
