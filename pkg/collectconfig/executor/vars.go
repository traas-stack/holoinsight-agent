package executor

import "github.com/traas-stack/holoinsight-agent/pkg/collectconfig"

type (
	varsProcessor struct {
		conf         *collectconfig.Vars
		varsCompiled []*varCompiled
	}
	varCompiled struct {
		conf  *collectconfig.Var
		elect XElect
	}
)

func parseVars(conf *collectconfig.Vars) (*varsProcessor, error) {
	if conf == nil {
		return nil, nil
	}

	varsCompiled := make([]*varCompiled, 0, len(conf.Vars))
	for _, var_ := range conf.Vars {
		elect, err := parseElect(var_.Elect)
		if err != nil {
			return nil, err
		}
		varsCompiled = append(varsCompiled, &varCompiled{
			conf:  var_,
			elect: elect,
		})
	}
	return &varsProcessor{
		conf:         conf,
		varsCompiled: varsCompiled,
	}, nil
}

func (p *varsProcessor) process(ctx *LogContext) (map[string]interface{}, error) {
	ret := make(map[string]interface{}, len(p.varsCompiled))
	for _, compiled := range p.varsCompiled {
		i, err := compiled.elect.Elect(ctx)
		if err != nil {
			return nil, err
		}
		ret[compiled.conf.Name] = i
	}
	return ret, nil
}
