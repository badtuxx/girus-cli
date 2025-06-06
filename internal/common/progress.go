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
	for _, lab := range p.Labs {
		if lab.Name == labName {
			return &lab
		}
	}
	return nil
}

func (p *Progress) SetLabStatus(labName string, status string) {
	lab := p.GetLab(labName)
	lab.Status = status
}
