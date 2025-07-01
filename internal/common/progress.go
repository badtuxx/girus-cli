package common

type Progress struct {
	Labs []Lab `yaml:"labs"`
}

type Lab struct {
	Name   string `yaml:"name"`
	Status string `yaml:"status"`
}

func (p *Progress) AddLab(labName string, status string) {
	p.Labs = append(p.Labs, Lab{
		Name:   labName,
		Status: status,
	})
}

func (p *Progress) GetLab(labName string) *Lab {
	for i := range p.Labs {
		if p.Labs[i].Name == labName {
			return &p.Labs[i]
		}
	}
	return nil
}

func (p *Progress) SetLabStatus(labName string, status string) {
	lab := p.GetLab(labName)
	if lab == nil {
		// Lab not found, handle the error gracefully
		return
	}
	lab.Status = status
}
