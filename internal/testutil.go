package internal

type MapListSorter []interface{}

func (s MapListSorter) Len() int {
	return len(s) / 2
}

func (s MapListSorter) Less(i, j int) bool {
	return s[i*2].(string) < s[j*2].(string)
}

func (s MapListSorter) Swap(i, j int) {
	s[i*2], s[j*2] = s[j*2], s[i*2]
	s[i*2+1], s[j*2+1] = s[j*2+1], s[i*2+1]
}
