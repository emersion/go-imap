package imapclient

import "strings"

// CustomAttribute holds the decoded value of a server-defined FETCH
// attribute. Exactly one payload field is populated by the constructors
// below; presence is detected through nil pointers (for scalars) and nil
// slices (for lists). Prefer the Get* getters, which return the typed value
// plus a presence boolean and avoid pointer dereferencing at call sites.
type CustomAttribute struct {
	Number      *uint64
	String      *string
	List        []CustomAttribute
	ListNumbers []uint64
	ListStrings []string
	// Raw is an escape hatch for shapes that don't fit the typed fields
	// above. Decoders that read the value verbatim (e.g. as an opaque token
	// or literal byte slice) put it here.
	Raw any
}

func NumberAttribute(n uint64) CustomAttribute {
	return CustomAttribute{Number: &n}
}

func StringAttribute(s string) CustomAttribute {
	return CustomAttribute{String: &s}
}

func ListAttribute(items []CustomAttribute) CustomAttribute {
	if items == nil {
		items = []CustomAttribute{}
	}
	return CustomAttribute{List: items}
}

func NumberListAttribute(nums []uint64) CustomAttribute {
	if nums == nil {
		nums = []uint64{}
	}
	return CustomAttribute{ListNumbers: nums}
}

func StringListAttribute(strs []string) CustomAttribute {
	if strs == nil {
		strs = []string{}
	}
	return CustomAttribute{ListStrings: strs}
}

func RawAttribute(v any) CustomAttribute {
	return CustomAttribute{Raw: v}
}

func (a CustomAttribute) IsNumber() bool     { return a.Number != nil }
func (a CustomAttribute) IsString() bool     { return a.String != nil }
func (a CustomAttribute) IsList() bool       { return a.List != nil }
func (a CustomAttribute) IsNumberList() bool { return a.ListNumbers != nil }
func (a CustomAttribute) IsStringList() bool { return a.ListStrings != nil }
func (a CustomAttribute) IsRaw() bool        { return a.Raw != nil }

// GetNumber returns the number value and true when Number is set; the zero
// value and false otherwise.
func (a CustomAttribute) GetNumber() (uint64, bool) {
	if a.Number == nil {
		return 0, false
	}
	return *a.Number, true
}

// GetString returns the string value and true when String is set.
func (a CustomAttribute) GetString() (string, bool) {
	if a.String == nil {
		return "", false
	}
	return *a.String, true
}

// GetList returns the heterogeneous list and true when List is set. An empty
// (but non-nil) list still reports ok=true.
func (a CustomAttribute) GetList() ([]CustomAttribute, bool) {
	return a.List, a.List != nil
}

func (a CustomAttribute) GetNumberList() ([]uint64, bool) {
	return a.ListNumbers, a.ListNumbers != nil
}

func (a CustomAttribute) GetStringList() ([]string, bool) {
	return a.ListStrings, a.ListStrings != nil
}

func (a CustomAttribute) GetRaw() (any, bool) {
	return a.Raw, a.Raw != nil
}

// CustomAttributes maps an upper-cased FETCH attribute name to its decoded
// value. Lookups via the helper methods below are case-insensitive.
type CustomAttributes map[string]CustomAttribute

// Get returns the CustomAttribute stored under name (matched
// case-insensitively) and whether it was present.
func (c CustomAttributes) Get(name string) (CustomAttribute, bool) {
	if c == nil {
		return CustomAttribute{}, false
	}
	if v, ok := c[name]; ok {
		return v, true
	}
	if v, ok := c[strings.ToUpper(name)]; ok {
		return v, true
	}
	for k, v := range c {
		if strings.EqualFold(k, name) {
			return v, true
		}
	}
	return CustomAttribute{}, false
}

// Has reports whether an attribute exists under name.
func (c CustomAttributes) Has(name string) bool {
	_, ok := c.Get(name)
	return ok
}

// GetNumber returns the Number payload of the named attribute. ok is true
// only when the attribute exists and was decoded as a number.
func (c CustomAttributes) GetNumber(name string) (uint64, bool) {
	v, ok := c.Get(name)
	if !ok {
		return 0, false
	}
	return v.GetNumber()
}

// HasNumber reports whether the named attribute exists and is a number.
func (c CustomAttributes) HasNumber(name string) bool {
	v, ok := c.Get(name)
	return ok && v.IsNumber()
}

// GetString returns the String payload of the named attribute.
func (c CustomAttributes) GetString(name string) (string, bool) {
	v, ok := c.Get(name)
	if !ok {
		return "", false
	}
	return v.GetString()
}

// HasString reports whether the named attribute exists and is a string.
func (c CustomAttributes) HasString(name string) bool {
	v, ok := c.Get(name)
	return ok && v.IsString()
}

// GetList returns the heterogeneous List payload of the named attribute.
func (c CustomAttributes) GetList(name string) ([]CustomAttribute, bool) {
	v, ok := c.Get(name)
	if !ok {
		return nil, false
	}
	return v.GetList()
}

// HasList reports whether the named attribute exists and is a list.
func (c CustomAttributes) HasList(name string) bool {
	v, ok := c.Get(name)
	return ok && v.IsList()
}

// GetNumberList returns the ListNumbers payload of the named attribute.
func (c CustomAttributes) GetNumberList(name string) ([]uint64, bool) {
	v, ok := c.Get(name)
	if !ok {
		return nil, false
	}
	return v.GetNumberList()
}

// HasNumberList reports whether the named attribute exists and is a list of
// numbers.
func (c CustomAttributes) HasNumberList(name string) bool {
	v, ok := c.Get(name)
	return ok && v.IsNumberList()
}

// GetStringList returns the ListStrings payload of the named attribute.
func (c CustomAttributes) GetStringList(name string) ([]string, bool) {
	v, ok := c.Get(name)
	if !ok {
		return nil, false
	}
	return v.GetStringList()
}

// HasStringList reports whether the named attribute exists and is a list of
// strings.
func (c CustomAttributes) HasStringList(name string) bool {
	v, ok := c.Get(name)
	return ok && v.IsStringList()
}

// GetRaw returns the Raw payload of the named attribute.
func (c CustomAttributes) GetRaw(name string) (any, bool) {
	v, ok := c.Get(name)
	if !ok {
		return nil, false
	}
	return v.GetRaw()
}

// HasRaw reports whether the named attribute exists and carries a Raw
// payload.
func (c CustomAttributes) HasRaw(name string) bool {
	v, ok := c.Get(name)
	return ok && v.IsRaw()
}
