package gosock

type param struct {
	key string
	val string
}

type Params []*param

func (p Params) Get(key string) (value string, ok bool) {
	for _, param := range p {
		if param.key == key {
			return param.val, true
		}
	}

	return "", false
}

func (p *Params) Add(key string, value string) {
	param := &param{key, value}
	*p = append(*p, param)
}

func (p *Params) Reset() {
	*p = (*p)[0:0]
}
