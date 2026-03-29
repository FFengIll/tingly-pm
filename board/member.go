package board

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// QueryMembers returns members filtered by optional query (name/label search) and memberType.
// - If query and memberType are both empty: returns all members.
// - If only memberType is set: filters by type.
// - If only query is set: case-insensitive substring search on name AND labels.
// - If both are set: search by query within the given type.
func QueryMembers(pmDir, query, memberType string) ([]Member, error) {
	mf, err := ReadMembers(pmDir)
	if err != nil {
		return nil, err
	}

	var result []Member
	q := strings.ToLower(query)

	for _, m := range mf.Members {
		// Filter by memberType if specified
		if memberType != "" && m.Type != memberType {
			continue
		}

		// If no query, include all (that passed type filter)
		if q == "" {
			result = append(result, m)
			continue
		}

		// Search name (case-insensitive substring)
		if strings.Contains(strings.ToLower(m.Name), q) {
			result = append(result, m)
			continue
		}

		// Search labels (case-insensitive substring)
		for _, label := range m.Labels {
			if strings.Contains(strings.ToLower(label), q) {
				result = append(result, m)
				break
			}
		}
	}
	return result, nil
}

// ListMembers returns members, optionally filtered by type.
// Deprecated: use QueryMembers instead.
func ListMembers(pmDir, memberType string) ([]Member, error) {
	return QueryMembers(pmDir, "", memberType)
}

// SearchMembers returns members matching any of the given labels.
// Deprecated: use QueryMembers instead.
func SearchMembers(pmDir string, labels string) ([]Member, error) {
	mf, err := ReadMembers(pmDir)
	if err != nil {
		return nil, err
	}

	if labels == "" {
		return mf.Members, nil
	}

	searchLabels := strings.Split(labels, ",")
	var result []Member
	for _, m := range mf.Members {
		for _, sl := range searchLabels {
			sl = strings.TrimSpace(sl)
			sl = strings.ToLower(sl)
			for _, ml := range m.Labels {
				if strings.Contains(strings.ToLower(ml), sl) {
					result = append(result, m)
					break
				}
			}
		}
	}
	return result, nil
}

// UpdateMember updates an existing member's type and/or labels
func UpdateMember(pmDir, name string, memberType string, labels []string) error {
	mf, err := ReadMembers(pmDir)
	if err != nil {
		return err
	}

	found := false
	for i, m := range mf.Members {
		if m.Name == name {
			found = true
			if memberType != "" {
				if memberType != "human" && memberType != "agent" {
					return fmt.Errorf("type must be 'human' or 'agent', got: %s", memberType)
				}
				mf.Members[i].Type = memberType
			}
			if labels != nil {
				mf.Members[i].Labels = labels
			}
			break
		}
	}

	if !found {
		return fmt.Errorf("member %s not found", name)
	}

	return WriteMembers(pmDir, mf)
}

// UpsertMember creates or updates a member
func UpsertMember(pmDir, name, memberType string, labels []string) error {
	mf, err := ReadMembers(pmDir)
	if err != nil {
		return err
	}

	// Check if member exists
	found := false
	for i, m := range mf.Members {
		if m.Name == name {
			found = true
			// Update existing member
			if memberType != "" {
				if memberType != "human" && memberType != "agent" {
					return fmt.Errorf("type must be 'human' or 'agent', got: %s", memberType)
				}
				mf.Members[i].Type = memberType
			}
			if labels != nil {
				mf.Members[i].Labels = labels
			}
			break
		}
	}

	// Create new member if not found
	if !found {
		if memberType == "" {
			return fmt.Errorf("member_type is required when creating new member %s", name)
		}
		if memberType != "human" && memberType != "agent" {
			return fmt.Errorf("type must be 'human' or 'agent', got: %s", memberType)
		}
		mf.Members = append(mf.Members, Member{
			Name:   name,
			Type:   memberType,
			Labels: labels,
		})
	}

	return WriteMembers(pmDir, mf)
}

// RemoveMember removes a member from the registry
func RemoveMember(pmDir, name string) error {
	mf, err := ReadMembers(pmDir)
	if err != nil {
		return err
	}

	found := false
	var newMembers []Member
	for _, m := range mf.Members {
		if m.Name != name {
			newMembers = append(newMembers, m)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("member %s not found", name)
	}

	mf.Members = newMembers
	return WriteMembers(pmDir, mf)
}
