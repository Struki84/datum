grep -n "fmt.Print" vendor/github.com/lokutor-ai/lokutor-orchestrator/pkg/orchestrator/managed_stream.go
sed -i '' 's/fmt\.Print/\/\/fmt.Print/g' vendor/github.com/lokutor-ai/lokutor-orchestrator/pkg/orchestrator/managed_stream.go
sed -i '' 's/\/\/fmt\.Print/fmt.Print/g' vendor/github.com/lokutor-ai/lokutor-orchestrator/pkg/orchestrator/managed_stream.go
