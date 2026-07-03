package collector

import "context"

// JobPlan describes a planned collection job (for --list-jobs).
type JobPlan struct {
	Name     string   `json:"name"`
	Args     []string `json:"args"`
	FileName string   `json:"file"`
	Optional bool     `json:"optional,omitempty"`
}

// ListJobs returns planned jobs without writing output or running collection.
func (s *Service) ListJobs(ctx context.Context) ([]JobPlan, error) {
	if err := s.initK8s(); err != nil {
		return nil, err
	}
	jobs, err := s.buildJobs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]JobPlan, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, JobPlan{
			Name:     j.Name,
			Args:     append([]string(nil), j.Args...),
			FileName: j.FileName,
			Optional: j.Optional,
		})
	}
	return out, nil
}
