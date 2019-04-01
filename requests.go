package screepsapi

import "github.com/hinshun/screepsapi/screepstype"

type AddObjectIntentRequest struct {
	Shard  string                   `json:"shard"`
	Room   string                   `json:"room"`
	ID     string                   `json:"_id"`
	Name   screepstype.ObjectIntent `json:"name"`
	Intent interface{}              `json:"intent"`
}

type CheckUniqueObjectNameRequest struct {
	Shard string                       `json:"shard"`
	Type  screepstype.UniqueObjectName `json:"type"`
	Name  string                       `json:"name"`
}

type ConsoleRequest struct {
        Shard string `json:"shard"`
	Expression string `json:"expression"`
}

type CreateFlagRequest struct {
	Shard          string            `json:"shard"`
	Room           string            `json:"room"`
	X              int               `json:"x"`
	Y              int               `json:"y"`
	Name           string            `json:"name"`
	Color          screepstype.Color `json:"color"`
	SecondaryColor screepstype.Color `json:"secondaryColor"`
}

type CreateConstructionRequest struct {
	Shard     string                `json:"shard"`
	Room      string                `json:"room"`
	X         int                   `json:"x"`
	Y         int                   `json:"y"`
	Structure screepstype.Structure `json:"type"`
}

type DestroyStructureIntentRequest struct {
	ID       string `json:"id"`
	RoomName string `json:"roomName"`
	User     string `json:"user"`
}

type GenUniqueObjectNameRequest struct {
	Shard string                       `json:"shard"`
	Type  screepstype.UniqueObjectName `json:"type"`
}

type MapStatsRequest struct {
	Shard    string               `json:"shard"`
	Rooms    []string             `json:"rooms"`
	StatName screepstype.StatName `json:"statName"`
}

type MemoryRequest struct {
	Shard string  `json:"shard"`
	Path  string  `json:"path"`
	Value *string `json:"value,omitempty"`
}

type MessagesSendRequest struct {
	Respondent string `json:"respondent"`
	Text       string `json:"text"`
}

type PlaceSpawnRequest struct {
	Shard string `json:"shard"`
	Room  string `json:"room"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Name  string `json:"name"`
}

type PushCodeRequest struct {
	Branch  string            `json:"branch,omitempty"`
	Modules map[string]string `json:"modules"`
}

type RemoveFlagRequest struct {
	Shard string `json:"shard"`
	Room  string `json:"room"`
	Name  string `json:"name"`
}

type SetActiveBranchRequest struct {
	ActiveName screepstype.ActiveName `json:"activeName"`
	Branch     string                 `json:"branch"`
}

type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateMemoryRequest struct {
	Shard   string `json:"shard"`
	Data    string `json:"data"`
	Segment int    `json:"segment"`
}
