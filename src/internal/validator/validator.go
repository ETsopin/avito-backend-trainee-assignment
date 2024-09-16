package validator

type Validator struct {
	Errors map[string]string
}

func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

func (v *Validator) AddError(key, value string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = value
	}
}

func (v *Validator) CheckError(ok bool, key, value string) {
	if !ok {
		v.AddError(key, value)
	}
}

func (v *Validator) StringNotEmpty(s string) bool {
	return len(s) > 0
}
