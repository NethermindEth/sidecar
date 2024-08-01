package parser

type ParsedTransaction struct {
	MethodName  string
	DecodedData string
	Logs        []*DecodedLog
}

type DecodedLog struct {
	LogIndex   uint64
	Address    string
	Arguments  []Argument
	EventName  string
	OutputData map[string]interface{}
}

type Argument struct {
	Name    string
	Type    string
	Value   interface{}
	Indexed bool
}
