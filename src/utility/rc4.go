package utility

//RC4Simple is a smplified version of rc4 encrypt algorithm
type RC4Simple struct {
	table []byte
}

//Init
func (r *RC4Simple) Init(key []byte) {

	t := make([]byte, 256)
	size := len(key)
	for i := 0; i < 256; i++ {
		t[i] = byte(i)
	}
	j := 0
	for i := 0; i < 256; i++ {
		j = (j + int(t[i]) + int(key[i%size])) % 256
		t[i], t[j] = t[j], t[i]
	}

	r.table = t
}

func (r *RC4Simple) Decrypt(src []byte) []byte {
	return r.Encrypt(src)
}

func (r *RC4Simple) Encrypt(src []byte) []byte {
	j := 0
	i := 0
	t := r.table
	for idx, v := range src {
		i = (i + 1) % 256
		j = (j + int(r.table[i])) % 256

		index := (int(t[i]) + int(t[j])) % 256
		src[idx] = t[index] ^ v
	}
	return src
}
