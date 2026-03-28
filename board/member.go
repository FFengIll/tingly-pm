package board

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Member represents a team member
type Member struct {
	Name   string   `json:"name"`
	Type   string   `json:"type"` // "human" or "agent"
	Labels []string `json:"labels,omitempty"`
}

// MembersFile is the structure of members.json
type MembersFile struct {
	Members []Member `json:"members"`
}

// ReadMembers reads members.json
func ReadMembers(pmDir string) (*MembersFile, error) {
	data, err := os.ReadFile(filepath.Join(pmDir, "members.json"))
	if err != nil {
		return &MembersFile{}, nil
	}
	var mf MembersFile
	if err := json.Unmarshal(data, &mf); err != nil {
		return nil, err
	}
	return &mf, nil
}

// WriteMembers writes members.json
func WriteMembers(pmDir string, mf *MembersFile) error {
	data, err := json.MarshalIndent(mf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(pmDir, "members.json"), append(data, '\n'), 0644)
}

// RegisterMember adds a new member
func RegisterMember(pmDir, name, memberType string, labels []string) error {
	mf, err := ReadMembers(pmDir)
	if err != nil {
		return err
	}

	for _, m := range mf.Members {
		if m.Name == name {
			return fmt.Errorf("member %s already exists", name)
		}
	}

	if memberType != "human" && memberType != "agent" {
		return fmt.Errorf("type must be 'human' or 'agent', got: %s", memberType)
	}

	mf.Members = append(mf.Members, Member{
		Name:   name,
		Type:   memberType,
		Labels: labels,
	})

	return WriteMembers(pmDir, mf)
}

// ListMembers returns members, optionally filtered by type
func ListMembers(pmDir, memberType string) ([]Member, error) {
	mf, err := ReadMembers(pmDir)
	if err != nil {
		return nil, err
	}

	if memberType == "" {
		return mf.Members, nil
	}

	var result []Member
	for _, m := range mf.Members {
		if m.Type == memberType {
			result = append(result, m)
		}
	}
	return result, nil
}
